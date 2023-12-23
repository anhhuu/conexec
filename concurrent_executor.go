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

// runTask executes tasks concurrently. It continuously dequeues tasks from the taskQueueChan
// and processes them in separate goroutines. It handles panics during task execution and
// reports errors to the responseChan.
func (concurrentExecutor *ConcurrentExecutor) runTask(ctx context.Context) {
	defer concurrentExecutor.waitgroup.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			task, ok := <-concurrentExecutor.taskQueueChan
			if !ok {
				return
			}
			defer func() {
				// Recover from panics and report errors to responseChan
				if r := recover(); r != nil {
					concurrentExecutor.mutex.Lock()
					concurrentExecutor.responseChan <- &TaskResponse{
						TaskID: task.ID,
						Value:  nil,
						Error:  fmt.Errorf("got panic, recover: %v", r),
					}
					concurrentExecutor.mutex.Unlock()
				}
			}()

			// Execute the task with provided arguments or without arguments
			var resp interface{}
			var err error
			if task.ExecutorArgs != nil && len(task.ExecutorArgs) > 0 {
				resp, err = task.Executor(ctx, task.ExecutorArgs...)
			} else {
				resp, err = task.Executor(ctx)
			}

			concurrentExecutor.responseChan <- &TaskResponse{
				TaskID: task.ID,
				Value:  resp,
				Error:  err,
			}
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

func (concurrentExecutor *ConcurrentExecutor) EnqueueTask(task Task) error {
	concurrentExecutor.mutex.Lock()
	if concurrentExecutor.closed {
		concurrentExecutor.mutex.Unlock()
		panic(ClosedPanicMsg)
	}

	if concurrentExecutor.isTaskQueueChanClosed {
		// Waiting for lastest running is finished and create a new queue (unlock mutex before waiting)
		concurrentExecutor.mutex.Unlock()
		concurrentExecutor.waitgroup.Wait()

		concurrentExecutor.mutex.Lock()
		concurrentExecutor.taskQueueChan = make(chan Task, concurrentExecutor.maxTaskQueueSize)
		// Init response channel
		concurrentExecutor.responseChan = make(chan *TaskResponse, concurrentExecutor.maxTaskQueueSize)
		concurrentExecutor.isTaskQueueChanClosed = false
	}
	concurrentExecutor.mutex.Unlock()

	select {
	case concurrentExecutor.taskQueueChan <- task:
		return nil
	default:
		return fmt.Errorf(FullQueueErr)
	}
}

func (concurrentExecutor *ConcurrentExecutor) StartExecution(ctx context.Context) {
	concurrentExecutor.mutex.Lock()
	if concurrentExecutor.closed {
		concurrentExecutor.mutex.Unlock()
		panic(ClosedPanicMsg)
	}

	// If it already run, return
	if concurrentExecutor.isTaskQueueChanClosed {
		concurrentExecutor.mutex.Unlock()
		return
	}
	concurrentExecutor.mutex.Unlock()

	// Reset internal state (Waiting for lastest running is finished before reset)
	concurrentExecutor.waitgroup.Wait()

	concurrentExecutor.mutex.Lock()
	concurrentExecutor.waitgroup = sync.WaitGroup{}
	concurrentExecutor.isTaskQueueChanClosed = false
	concurrentExecutor.closeResponseChanOnce = sync.Once{}
	concurrentExecutor.closeTaskQueueChanOnce = sync.Once{}
	concurrentExecutor.mutex.Unlock()

	// Run task
	for i := 0; i < concurrentExecutor.maxConcurrentTasks; i++ {
		concurrentExecutor.waitgroup.Add(1)
		go concurrentExecutor.runTask(ctx)
	}

	// After run all tasks in goroutine, taskQueueChan channel should be closed to block publish new task into channel
	// and avoid deadlock on goroutine (when consume all of tasks in queue)
	concurrentExecutor.closeTaskQueueChanOnce.Do(func() {
		concurrentExecutor.mutex.Lock()
		concurrentExecutor.isTaskQueueChanClosed = true
		concurrentExecutor.mutex.Unlock()
		close(concurrentExecutor.taskQueueChan)
	})
}

func (concurrentExecutor *ConcurrentExecutor) WaitForCompletionAndGetResponse() map[string]*TaskResponse {
	concurrentExecutor.mutex.Lock()
	// Haven't ran yet
	if !concurrentExecutor.isTaskQueueChanClosed {
		concurrentExecutor.mutex.Unlock()
		return make(map[string]*TaskResponse, 0)
	}

	if concurrentExecutor.closed {
		concurrentExecutor.mutex.Unlock()
		panic(ClosedPanicMsg)
	}
	concurrentExecutor.mutex.Unlock()

	concurrentExecutor.waitgroup.Wait()

	// After wait all running tasks in goroutine finished, responseChan channel should be closed to block publish new resp into channel
	// and avoid deadlock on goroutine (when reading all of responses in channel)
	concurrentExecutor.closeResponseChanOnce.Do(func() {
		close(concurrentExecutor.responseChan)
	})

	resp := make(map[string]*TaskResponse, 0)
	for {
		r, ok := <-concurrentExecutor.responseChan
		if !ok {
			break
		}

		resp[r.TaskID] = r
	}

	return resp
}

// Close must be called after WaitForCompletionAndGetResponse, if not, will panic, suggest use: defer Close()
func (concurrentExecutor *ConcurrentExecutor) Close() {
	concurrentExecutor.closeTaskQueueChanOnce.Do(func() {
		concurrentExecutor.mutex.Lock()
		concurrentExecutor.isTaskQueueChanClosed = true
		concurrentExecutor.mutex.Unlock()
		close(concurrentExecutor.taskQueueChan)
	})
	concurrentExecutor.closeResponseChanOnce.Do(func() {
		close(concurrentExecutor.responseChan)
	})

	concurrentExecutor.mutex.Lock()
	concurrentExecutor.closed = true
	concurrentExecutor.mutex.Unlock()
}
