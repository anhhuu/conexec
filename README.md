# Concurrent Executor for Asynchronous Task Execution

Package name: `conexe`

## Overview

The `conexe` package provides a Concurrent Executor that facilitates the concurrent execution of multiple tasks, managing its own task queue. It is designed to handle asynchronous task execution with controlled concurrency and a task queue to ensure efficient resource utilization.

## Features

- **Task Execution:** Execute tasks concurrently while respecting the specified maximum concurrent task limit.
- **Task Queue:** Manage a task queue to handle tasks that exceed the current concurrency limit.
- **Error Handling:** Capture and report errors during task execution.
- **Task Response:** Retrieve responses and errors for each completed task.

## Example

``` go
// Initialize Concurrent Executor
executor := NewConcurrentExecutor(5, 10)

// Enqueue tasks
task1 := Task{ID: "1", Executor: myTaskExecutorFunc, ExecutorArgs: []interface{}{"arg1"}}
executor.EnqueueTask(task1)

// Start task execution
executor.StartExecution(context.Background())

// Wait for completion and get responses
responses := executor.WaitForCompletionAndGetResponse()

// Close the executor
executor.Close()
```

## Usage

### Task Structure

```go
type Task struct {
    ID           string
    Executor     TaskExecutor
    ExecutorArgs []interface{}
}

type TaskExecutor func(ctx context.Context, args ...interface{}) (interface{}, error)
```

The `Task` struct represents a task with a unique identifier (`ID`), an executor function (`Executor`), and optional executor arguments (ExecutorArgs). The executor function takes a context and variable arguments and returns a value and an error.

### Concurrent Executor Initialization

```go
func NewConcurrentExecutor(maxConcurrentTasks, maxTaskQueueSize int) *ConcurrentExecutor
```

Initialize a new Concurrent Executor with the specified maximum concurrent tasks and task queue size.

### Enqueue Task

```go
func (ce *ConcurrentExecutor) EnqueueTask(task Task) error
```

Enqueue a task for execution. If the task queue is full, an error is returned.

### Start Execution

```go
func (ce *ConcurrentExecutor) StartExecution(ctx context.Context)
```

Start the execution of tasks in a concurrent manner. It initiates goroutines to process tasks up to the specified maximum concurrency.

### Wait for Completion and Get Responses

```go
func (ce *ConcurrentExecutor) WaitForCompletionAndGetResponse() map[string]*TaskResponse
```

Wait for all tasks to complete and retrieve responses along with errors for each task.

### Close

```go
func (ce *ConcurrentExecutor) Close()
```

Close the task queue and response channels. This method should be called after `WaitForCompletionAndGetResponse` to ensure proper cleanup. Usage of defer `ce.Close()` is recommended.

## Important Notes

- Ensure to call `Close` after using the executor to release resources properly.
- Use defer `executor.Close()` to automatically close the executor after task execution.

## License

This package is distributed under the **MIT License**. Feel free to use, modify, and distribute it as needed. Contributions are welcome!
