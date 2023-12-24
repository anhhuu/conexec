# conexec #

[![Release](https://img.shields.io/github/release/anhhuu/conexec.svg?color=brightgreen)](https://github.com/anhhuu/conexec/releases/)
[![build](https://github.com/anhhuu/conexec/actions/workflows/go.yml/badge.svg?branch=main)](https://github.com/anhhuu/conexec/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/anhhuu/conexec/branch/main/graph/badge.svg)](https://codecov.io/gh/anhhuu/conexec)
[![GoReportCard](https://goreportcard.com/badge/github.com/anhhuu/conexec)](https://goreportcard.com/report/github.com/anhhuu/conexec)
[![Go Reference](https://pkg.go.dev/badge/github.com/anhhuu/conexec.svg)](https://pkg.go.dev/github.com/anhhuu/conexec)
[![MIT License](https://img.shields.io/badge/License-MIT-green.svg)](https://github.com/anhhuu/conexec/blob/main/LICENSE)

Package `anhhuu/conexec` (**Concurrent Executor for Asynchronous Task Execution**) provides a concurrent executor that facilitates the concurrent execution of multiple tasks (can config the maximum value), managing its own task queue. It is designed to handle asynchronous task execution with controlled concurrency and a task queue to ensure efficient resource utilization.

## Installation ##

```bash
go get github.com/anhhuu/conexec
```

## Example ##

``` go
package main

import (
    "context"
    "fmt"

    "github.com/anhhuu/conexec"
)

func main() {
    // Initialize Concurrent Executor with default config
    executor := conexec.NewConcurrentExecutorBuilder().Build()
    defer executor.Close()

    // Enqueue a task
    demoTask := conexec.Task{
        ID: "demo-task",
        ExecutorArgs: []interface{}{"first-arg"},
        Executor: func(ctx context.Context, args ...interface{}) (interface{}, error) {
            firstArg, ok := args[0].(string)
            if !ok {
                return "", fmt.Errorf("parse error")
            }

            // ... Logic of your executor

            return firstArg, nil
        },
    }

    executor.EnqueueTask(demoTask)

    // Start task execution
    executor.StartExecution(context.Background())

    // Wait for completion and get responses
    responses := executor.WaitForCompletionAndGetResponse()

    // Do something with `responses`
}
```

## Features ##

- **Task Execution:** Execute tasks concurrently while respecting the specified maximum concurrent task limit.
- **Task Queue:** Manage a task queue to handle tasks that exceed the current concurrency limit.
- **Error Handling:** Capture and report errors/panics during task execution.
- **Task Response:** Retrieve responses and errors for each completed task.

## Usage ##

```go
import (
    "github.com/anhhuu/conexec"
)
```

### Task Structure ###

```go
type Task struct {
    ID           string
    Executor     TaskExecutor
    ExecutorArgs []interface{}
}

type TaskExecutor func(ctx context.Context, args ...interface{}) (interface{}, error)
```

`Task` struct represents a task with a unique identifier (`ID`), an executor function (`Executor`) with custom params, and optional executor arguments (`ExecutorArgs`). The executor function takes a context and variable arguments and returns a value and an error.

### Concurrent Executor Initialization ###

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

### Enqueue Task ###

```go
func (concurrentExecutor *ConcurrentExecutor) EnqueueTask(task Task) error
```

Enqueue a task for execution. If the task queue is full, an error is returned. If the executor is closed before calling this method, a panic with `ClosedPanicMsg` is triggered.

### Start Execution ###

```go
func (concurrentExecutor *ConcurrentExecutor) StartExecution(ctx context.Context)
```

`StartExecution` initiates the concurrent execution of tasks. It takes a context as an argument to support cancellation of the task execution.

```go
{
    ...
    ctx, cancel := context.WithCancel(context.Background())
    task.StartExecution(ctx)
    time.Sleep(time.Duration(1) * time.Second)

    cancel() // All executions will be canceled after this is called
    ...
}
```
If the executor is closed before calling this method, a panic with `ClosedPanicMsg` is triggered.

### Wait for Completion and Get Responses ###

```go
func (concurrentExecutor *ConcurrentExecutor) WaitForCompletionAndGetResponse() map[string]*TaskResponse
```

`WaitForCompletionAndGetResponse` waits for the concurrent execution of tasks to complete and retrieves the responses for each task. It returns a map where the keys are task IDs, and the values are `TaskResponse` structures containing the results or errors of the tasks. If the executor is closed before calling this method, a panic with `ClosedPanicMsg` is triggered. If the executor has not been run yet, an empty map is returned. This method blocks until all tasks have completed their execution.

### Close ###

```go
func (concurrentExecutor *ConcurrentExecutor) Close()
```

Close the task queue and response channels. This method should be called after `EnqueueTask`/`StartExecution`/`WaitForCompletionAndGetResponse` to ensure proper cleanup. Usage of defer `concurrentExecutor.Close()` is recommended.

## Limitation ##

- Can not Enqueue a new task for an executor when it's running, need to wait for the executor is finished.

## License ##

This package is distributed under the **MIT License**. Feel free to use, modify, and distribute it as needed. Contributions are welcome!
