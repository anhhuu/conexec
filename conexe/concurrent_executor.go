package conexe

import (
	"context"
	"fmt"
	"sync"
)

/*
// If ExecutorArgs is valid, it will be passed into Executor.

	{
		...
		task.Executor(ctx, task.ExecutorArgs...)
	}

// Then in Executor function, we can get args and parse type to use to call another func (avoid Captured Closure):

	{
		...
		task.Executor = func(ctx context.Context, args ...interface{}) (interface{}, error) {
			arg0, ok := args[0].(type)
			if !ok {
				return "", fmt.Errorf("parse error")
			}

			// Do something with arg0

			return arg0, nil
		}
	}
*/
type Task struct {
	ID           string
	Executor     TaskExecutor
	ExecutorArgs []interface{}
}

type TaskExecutor func(ctx context.Context, args ...interface{}) (interface{}, error)

type TaskResponse struct {
	TaskID string
	Value  interface{}
	Error  error
}

type ConcurrentExecutor struct {
	maxTaskQueueSize       int
	maxConcurrentTasks     int
	waitgroup              sync.WaitGroup
	mutex                  sync.Mutex
	taskQueueChan          chan Task
	responseChan           chan *TaskResponse
	closeTaskQueueChanOnce sync.Once
	closeResponseChanOnce  sync.Once
	isTaskQueueChanClosed  bool
	closed                 bool
}

func (tr *ConcurrentExecutor) runTask(ctx context.Context) {
	defer tr.waitgroup.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			task, ok := <-tr.taskQueueChan
			if !ok {
				return
			}
			defer func() {
				if r := recover(); r != nil {
					tr.mutex.Lock()
					tr.responseChan <- &TaskResponse{
						TaskID: task.ID,
						Value:  nil,
						Error:  fmt.Errorf("got panic, recover: %v", r),
					}
					tr.mutex.Unlock()
				}
			}()

			var resp interface{}
			var err error
			if task.ExecutorArgs != nil && len(task.ExecutorArgs) > 0 {
				resp, err = task.Executor(ctx, task.ExecutorArgs...)
			} else {
				resp, err = task.Executor(ctx)
			}

			tr.mutex.Lock()
			tr.responseChan <- &TaskResponse{
				TaskID: task.ID,
				Value:  resp,
				Error:  err,
			}
			tr.mutex.Unlock()
		}
	}
}

/*
The ConcurrentExecutor facilitates the concurrent execution of multiple tasks by managing its own task queue. When a task is received, it is enqueued using the EnqueueTask method. The executor adheres to the following rules:

  - If there are currently N tasks running concurrently, the incoming task is not processed.
  - If the number of concurrently running tasks is less than N, the executor initiates the execution of the task.

Upon completion of a task from the queue:

  - If additional tasks are still available in the queue, they are dequeued and executed.
  - If no tasks are currently running, and the queue is empty, a message is generated indicating that all tasks have been completed.
*/
func NewConcurrentExecutor(maxConcurrentTasks, maxTaskQueueSize int) *ConcurrentExecutor {
	return &ConcurrentExecutor{
		maxTaskQueueSize:      maxTaskQueueSize,
		maxConcurrentTasks:    maxConcurrentTasks,
		taskQueueChan:         make(chan Task, maxTaskQueueSize),
		responseChan:          make(chan *TaskResponse, maxTaskQueueSize),
		isTaskQueueChanClosed: false,
		closed:                false,
	}
}

func (tr *ConcurrentExecutor) EnqueueTask(task Task) error {
	tr.mutex.Lock()
	if tr.closed {
		tr.mutex.Unlock()
		panic(ClosedPanicMsg)
	}

	if tr.isTaskQueueChanClosed {
		// Waiting for lastest running is finished and create a new queue (unlock mutex before waiting)
		tr.mutex.Unlock()
		tr.waitgroup.Wait()

		tr.mutex.Lock()
		tr.taskQueueChan = make(chan Task, tr.maxTaskQueueSize)
		// Init response channel
		tr.responseChan = make(chan *TaskResponse, tr.maxTaskQueueSize)
		tr.isTaskQueueChanClosed = false
	}
	tr.mutex.Unlock()

	select {
	case tr.taskQueueChan <- task:
		return nil
	default:
		return fmt.Errorf(FullQueueErr)
	}
}

func (tr *ConcurrentExecutor) StartExecution(ctx context.Context) {
	tr.mutex.Lock()
	if tr.closed {
		tr.mutex.Unlock()
		panic(ClosedPanicMsg)
	}

	// If it already run, return
	if tr.isTaskQueueChanClosed {
		tr.mutex.Unlock()
		return
	}
	tr.mutex.Unlock()

	// Reset internal state (Waiting for lastest running is finished before reset)
	tr.waitgroup.Wait()

	tr.mutex.Lock()
	tr.waitgroup = sync.WaitGroup{}
	tr.isTaskQueueChanClosed = false
	tr.closeResponseChanOnce = sync.Once{}
	tr.closeTaskQueueChanOnce = sync.Once{}
	tr.mutex.Unlock()

	// Run task
	for i := 0; i < tr.maxConcurrentTasks; i++ {
		tr.waitgroup.Add(1)
		go tr.runTask(ctx)
	}

	// After run all tasks in goroutine, taskQueueChan channel should be closed to block publish new task into channel
	// and avoid deadlock on goroutine (when consume all of tasks in queue)
	tr.closeTaskQueueChanOnce.Do(func() {
		tr.mutex.Lock()
		tr.isTaskQueueChanClosed = true
		tr.mutex.Unlock()
		close(tr.taskQueueChan)
	})
}

func (tr *ConcurrentExecutor) WaitForCompletionAndGetResponse() map[string]*TaskResponse {
	tr.mutex.Lock()
	// Haven't ran yet
	if !tr.isTaskQueueChanClosed {
		tr.mutex.Unlock()
		return make(map[string]*TaskResponse, 0)
	}

	if tr.closed {
		tr.mutex.Unlock()
		panic(ClosedPanicMsg)
	}
	tr.mutex.Unlock()

	tr.waitgroup.Wait()

	// After wait all running tasks in goroutine finished, responseChan channel should be closed to block publish new resp into channel
	// and avoid deadlock on goroutine (when reading all of responses in channel)
	tr.closeResponseChanOnce.Do(func() {
		close(tr.responseChan)
	})

	resp := make(map[string]*TaskResponse, 0)
	for {
		r, ok := <-tr.responseChan
		if !ok {
			break
		}

		resp[r.TaskID] = r
	}

	return resp
}

// Close must be called after WaitForCompletionAndGetResponse, if not, will panic, suggest use: defer Close()
func (tr *ConcurrentExecutor) Close() {
	tr.closeTaskQueueChanOnce.Do(func() {
		tr.mutex.Lock()
		tr.isTaskQueueChanClosed = true
		tr.mutex.Unlock()
		close(tr.taskQueueChan)
	})
	tr.closeResponseChanOnce.Do(func() {
		close(tr.responseChan)
	})

	tr.mutex.Lock()
	tr.closed = true
	tr.mutex.Unlock()
}
