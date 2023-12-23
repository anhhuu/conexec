package conexec

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

func (ce *ConcurrentExecutor) runTask(ctx context.Context) {
	defer ce.waitgroup.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			task, ok := <-ce.taskQueueChan
			if !ok {
				return
			}
			defer func() {
				if r := recover(); r != nil {
					ce.mutex.Lock()
					ce.responseChan <- &TaskResponse{
						TaskID: task.ID,
						Value:  nil,
						Error:  fmt.Errorf("got panic, recover: %v", r),
					}
					ce.mutex.Unlock()
				}
			}()

			var resp interface{}
			var err error
			if task.ExecutorArgs != nil && len(task.ExecutorArgs) > 0 {
				resp, err = task.Executor(ctx, task.ExecutorArgs...)
			} else {
				resp, err = task.Executor(ctx)
			}

			ce.mutex.Lock()
			ce.responseChan <- &TaskResponse{
				TaskID: task.ID,
				Value:  resp,
				Error:  err,
			}
			ce.mutex.Unlock()
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

func (ce *ConcurrentExecutor) EnqueueTask(task Task) error {
	ce.mutex.Lock()
	if ce.closed {
		ce.mutex.Unlock()
		panic(ClosedPanicMsg)
	}

	if ce.isTaskQueueChanClosed {
		// Waiting for lastest running is finished and create a new queue (unlock mutex before waiting)
		ce.mutex.Unlock()
		ce.waitgroup.Wait()

		ce.mutex.Lock()
		ce.taskQueueChan = make(chan Task, ce.maxTaskQueueSize)
		// Init response channel
		ce.responseChan = make(chan *TaskResponse, ce.maxTaskQueueSize)
		ce.isTaskQueueChanClosed = false
	}
	ce.mutex.Unlock()

	select {
	case ce.taskQueueChan <- task:
		return nil
	default:
		return fmt.Errorf(FullQueueErr)
	}
}

func (ce *ConcurrentExecutor) StartExecution(ctx context.Context) {
	ce.mutex.Lock()
	if ce.closed {
		ce.mutex.Unlock()
		panic(ClosedPanicMsg)
	}

	// If it already run, return
	if ce.isTaskQueueChanClosed {
		ce.mutex.Unlock()
		return
	}
	ce.mutex.Unlock()

	// Reset internal state (Waiting for lastest running is finished before reset)
	ce.waitgroup.Wait()

	ce.mutex.Lock()
	ce.waitgroup = sync.WaitGroup{}
	ce.isTaskQueueChanClosed = false
	ce.closeResponseChanOnce = sync.Once{}
	ce.closeTaskQueueChanOnce = sync.Once{}
	ce.mutex.Unlock()

	// Run task
	for i := 0; i < ce.maxConcurrentTasks; i++ {
		ce.waitgroup.Add(1)
		go ce.runTask(ctx)
	}

	// After run all tasks in goroutine, taskQueueChan channel should be closed to block publish new task into channel
	// and avoid deadlock on goroutine (when consume all of tasks in queue)
	ce.closeTaskQueueChanOnce.Do(func() {
		ce.mutex.Lock()
		ce.isTaskQueueChanClosed = true
		ce.mutex.Unlock()
		close(ce.taskQueueChan)
	})
}

func (ce *ConcurrentExecutor) WaitForCompletionAndGetResponse() map[string]*TaskResponse {
	ce.mutex.Lock()
	// Haven't ran yet
	if !ce.isTaskQueueChanClosed {
		ce.mutex.Unlock()
		return make(map[string]*TaskResponse, 0)
	}

	if ce.closed {
		ce.mutex.Unlock()
		panic(ClosedPanicMsg)
	}
	ce.mutex.Unlock()

	ce.waitgroup.Wait()

	// After wait all running tasks in goroutine finished, responseChan channel should be closed to block publish new resp into channel
	// and avoid deadlock on goroutine (when reading all of responses in channel)
	ce.closeResponseChanOnce.Do(func() {
		close(ce.responseChan)
	})

	resp := make(map[string]*TaskResponse, 0)
	for {
		r, ok := <-ce.responseChan
		if !ok {
			break
		}

		resp[r.TaskID] = r
	}

	return resp
}

// Close must be called after WaitForCompletionAndGetResponse, if not, will panic, suggest use: defer Close()
func (ce *ConcurrentExecutor) Close() {
	ce.closeTaskQueueChanOnce.Do(func() {
		ce.mutex.Lock()
		ce.isTaskQueueChanClosed = true
		ce.mutex.Unlock()
		close(ce.taskQueueChan)
	})
	ce.closeResponseChanOnce.Do(func() {
		close(ce.responseChan)
	})

	ce.mutex.Lock()
	ce.closed = true
	ce.mutex.Unlock()
}
