package conexec

import (
	"context"
	"sync"

	"github.com/pkg/errors"
)

/*
If `ExecutorArgs` is valid, it will be passed into `Executor` func.

	{
		...
		task.Executor(ctx, task.ExecutorArgs...)
	}

Then in `Executor` function, we can get args and parse type to use to call another func (avoid Captured Closure):

	{
		...
		task.Executor = func(ctx context.Context, args ...interface{}) (interface{}, error) {
			firstArg, ok := args[0].(type)
			if !ok {
				return "", errors.Errorf("parse error")
			}

			// Do something with firstArg ...

			return firstArg, nil
		}
	}
*/
type Task struct {
	ID           string
	Executor     TaskExecutor
	ExecutorArgs []interface{}
}

// `TaskExecutor` is a function signature representing an executable task.
// It takes a context and variable arguments (args) and returns the result
// of the task execution along with any encountered error.
// The context can be used for cancellation and deadline propagation.
type TaskExecutor func(ctx context.Context, args ...interface{}) (interface{}, error)

// `TaskResponse` represents the result of a task execution.
// It includes the `TaskID` to uniquely identify the corresponding task,
// the `Value` holding the result of the task execution, and the `Error` containing
// any error encountered during execution.
type TaskResponse struct {
	TaskID string

	// `Value` holds the result of the task execution.
	Value interface{}

	// `Error` contains any error encountered during the task execution.
	Error error
}

/*
`ConcurrentExecutor` struct facilitates the concurrent execution of multiple tasks by managing its own task queue.
It follows specific rules for task execution:
  - If there are currently N tasks running concurrently, incoming tasks are not processed.
  - If the number of concurrently running tasks is less than N, the executor initiates the execution of the task.

Upon completion of a task from the queue:
  - If additional tasks are available, they are dequeued and executed.
  - If no tasks are currently running, and the queue is empty, a message indicates that all tasks have been completed.
*/
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

// `runTask` executes tasks concurrently. It continuously dequeues tasks from the `taskQueueChan`
// and processes them in separate goroutines. It handles panics during task execution and
// reports errors to the `responseChan`.
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
				// Recover from panics and report errors to `responseChan`
				if r := recover(); r != nil {
					// Handle panic by reporting an error to responseChan
					concurrentExecutor.responseChan <- &TaskResponse{
						TaskID: task.ID,
						Value:  nil,
						Error:  errors.Errorf("got panic, recover: %v", r),
					}

					// This goroutine is destroyed because panic -> Create a new goroutine to replace the one that panicked,
					// supporting handling of another task
					concurrentExecutor.waitgroup.Add(1)
					go concurrentExecutor.runTask(ctx)
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

			// Send the response to the `responseChan`
			concurrentExecutor.responseChan <- &TaskResponse{
				TaskID: task.ID,
				Value:  resp,
				Error:  err,
			}
		}
	}
}

// `NewConcurrentExecutor` creates a new `ConcurrentExecutor` with the specified maximum concurrent tasks
// and maximum task queue size.
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

// `EnqueueTask` adds a task to the concurrent executor's task queue. It panics with `ClosedPanicMsg`
// if `EnqueueTask` is called on a closed executor. It returns an error if the task cannot be added,
// for example, if the task queue is full (`FullQueueErr`).
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
		return errors.Errorf(FullQueueErr)
	}
}

/*
`StartExecution` initiates the concurrent execution of tasks. Will panic if we call `StartExecution` of a closed executor.
It takes a context as an argument to support cancellation of the task execution.

	{
		...
		ctx, cancel := context.WithCancel(context.Background())
		task.StartExecution(ctx)
		time.Sleep(time.Duration(1) * time.Second)

		cancel() // All executions will be canceled after this is called
		...
	}
*/
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

	// After run all tasks in goroutine, `taskQueueChan`` channel should be closed to block publish new task into channel
	// and avoid deadlock on goroutine (when consume all of tasks in queue)
	concurrentExecutor.closeTaskQueueChanOnce.Do(func() {
		concurrentExecutor.mutex.Lock()
		concurrentExecutor.isTaskQueueChanClosed = true
		concurrentExecutor.mutex.Unlock()
		close(concurrentExecutor.taskQueueChan)
	})
}

// `WaitForCompletionAndGetResponse` waits for the concurrent execution of tasks to complete
// and retrieves the responses for each task. It returns a map where the keys are task IDs,
// and the values are `TaskResponse` structures containing the results or errors of the tasks.
// If the executor is closed before calling this method, a panic with `ClosedPanicMsg` is triggered.
// If the executor has not been run yet, an empty map is returned.
// This method blocks until all tasks have completed their execution.
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

	// After wait all running tasks in goroutine finished, `responseChan` channel should be closed to block publish new resp into channel
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

// `Close` should be called after `EnqueueTask`/`StartExecution`/`WaitForCompletionAndGetResponse`. If not, a panic with `ClosedPanicMsg` is triggered.
// It is suggested to use `defer Close()` to ensure proper cleanup.
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
