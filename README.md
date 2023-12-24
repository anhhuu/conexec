# conexec #

[![build](https://github.com/anhhuu/conexec/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/anhhuu/conexec/actions/workflows/go.yml)
[![Codecov](https://codecov.io/gh/anhhuu/conexec/branch/main/graph/badge.svg)](https://codecov.io/gh/anhhuu/conexec)
[![GoReportCard](https://goreportcard.com/badge/github.com/anhhuu/conexec)](https://goreportcard.com/report/github.com/anhhuu/conexec)
[![MIT License](https://img.shields.io/badge/License-MIT-green.svg)](https://github.com/anhhuu/conexec/blob/main/LICENSE)

Package `anhhuu/conexec` (**Concurrent Executor for Asynchronous Task Execution**) provides a Concurrent Executor that facilitates the concurrent execution of multiple tasks, managing its own task queue. It is designed to handle asynchronous task execution with controlled concurrency and a task queue to ensure efficient resource utilization.

## Installation

```bash
go get github.com/anhhuu/conexec
```

## Example

``` go
package main

import (
    "context"
    "fmt"

    "github.com/anhhuu/conexec"
)

func main() {
    // Initialize Concurrent Executor
    executor := conexec.NewConcurrentExecutorBuilder().Build()
    defer executor.Close()

    // Enqueue tasks
    task1 := conexec.Task{
        ID: "1",
        Executor: func(ctx context.Context, args ...interface{}) (interface{}, error) {
            arg0, ok := args[0].(string)
            if !ok {
                return "", fmt.Errorf("parse error")
            }
            return arg0, nil
        },
        ExecutorArgs: []interface{}{"arg0"},
    }

    executor.EnqueueTask(task1)

    // Start task execution
    executor.StartExecution(context.Background())

    // Wait for completion and get responses
    responses := executor.WaitForCompletionAndGetResponse()

    // Do something with `responses`
}
```

## Features

- **Task Execution:** Execute tasks concurrently while respecting the specified maximum concurrent task limit.
- **Task Queue:** Manage a task queue to handle tasks that exceed the current concurrency limit.
- **Error Handling:** Capture and report errors/panics during task execution.
- **Task Response:** Retrieve responses and errors for each completed task.

## Usage

```go
import (
    "github.com/anhhuu/conexec"
)
```

### Task Structure

```go
type Task struct {
    ID           string
    Executor     TaskExecutor
    ExecutorArgs []interface{}
}

type TaskExecutor func(ctx context.Context, args ...interface{}) (interface{}, error)
```

The `Task` struct represents a task with a unique identifier (`ID`), an executor function (`Executor`), and optional executor arguments (`ExecutorArgs`). The executor function takes a context and variable arguments and returns a value and an error.

### Concurrent Executor Initialization

```go
func NewConcurrentExecutor(maxConcurrentTasks, maxTaskQueueSize int) *ConcurrentExecutor
```

Initialize a new Concurrent Executor with the specified maximum concurrent tasks and task queue size.
Or simple to use builder:

```go
concurrentExecutor := conexec.NewConcurrentExecutorBuilder().
    WithMaxTaskQueueSize(defautMaxTaskQueueSize).
    WithMaxConcurrentTasks(defaultMaxConcurrentTasks).
    Build()
```

### Enqueue Task

```go
func (concurrentExecutor *ConcurrentExecutor) EnqueueTask(task Task) error
```

Enqueue a task for execution. If the task queue is full, an error is returned.

### Start Execution

```go
func (concurrentExecutor *ConcurrentExecutor) StartExecution(ctx context.Context)
```

Start the execution of tasks in a concurrent manner. It initiates goroutines to process tasks up to the specified maximum concurrency.

### Wait for Completion and Get Responses

```go
func (concurrentExecutor *ConcurrentExecutor) WaitForCompletionAndGetResponse() map[string]*TaskResponse
```

Wait for all tasks to complete and retrieve responses along with errors for each task.

### Close

```go
func (concurrentExecutor *ConcurrentExecutor) Close()
```

Close the task queue and response channels. This method should be called after `WaitForCompletionAndGetResponse` to ensure proper cleanup. Usage of defer `concurrentExecutor.Close()` is recommended.

## Important Notes

- Ensure to call `Close` after using the executor to release resources properly.
- Use defer `executor.Close()` to automatically close the executor after task execution.

## License

This package is distributed under the **MIT License**. Feel free to use, modify, and distribute it as needed. Contributions are welcome!
