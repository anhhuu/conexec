package conexec

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const (
	numberOfTestingTasks = 50
)

func executorForTest(ctx context.Context, args ...interface{}) (interface{}, error) {
	if len(args) != 3 {
		return "", errors.Errorf("func dummyExecutor need 3 args, got %d", len(args))
	}

	taskID, ok := args[0].(string)
	if !ok {
		return "", errors.Errorf("parse taskID error")
	}

	isReturnError, ok := args[1].(bool)
	if !ok {
		return "", errors.Errorf("parse isReturnError error")
	}

	// Because expectedErr arg can NULL, so need a check to avoid panic
	var expectedError error
	if args[2] != nil {
		if expectedError, ok = args[2].(error); !ok {
			return "", errors.Errorf("parse expectedError error")
		}
	}

	// Simulating task execution
	time.Sleep(5 * time.Millisecond)

	if isReturnError {
		return "", expectedError
	}
	return taskID, nil
}

func executorForPanicTest(ctx context.Context, args ...interface{}) (interface{}, error) {
	if len(args) != 3 {
		return "", errors.Errorf("func dummyExecutor need 3 args, got %d", len(args))
	}

	taskID, ok := args[0].(string)
	if !ok {
		return "", errors.Errorf("parse taskID error")
	}

	isPanic, ok := args[1].(bool)
	if !ok {
		return "", errors.Errorf("parse isPanic error")
	}

	expectedPanicMsg, ok := args[2].(string)
	if !ok {
		return "", errors.Errorf("parse expectedPanicMsg error")
	}

	// Simulating task execution
	time.Sleep(5 * time.Millisecond)

	if isPanic {
		panic(expectedPanicMsg)
	}
	return taskID, nil
}

func getTaskIDForTest(index int) string {
	return "Task_" + strconv.Itoa(index)
}

func TestConcurrentExecutor_SingleRun(t *testing.T) {
	t.Parallel()
	test := assert.New(t)

	// Create a ConcurrentExecutor
	concurrentExecutor := NewConcurrentExecutorBuilder().
		WithMaxTaskQueueSize(defautMaxTaskQueueSize).
		WithMaxConcurrentTasks(defaultMaxConcurrentTasks).
		Build()

	// Adding tasks
	for i := 1; i <= numberOfTestingTasks; i++ {
		taskID := getTaskIDForTest(i)
		task := Task{
			ID:           taskID,
			ExecutorArgs: []interface{}{taskID, false, nil},
			Executor:     executorForTest,
		}
		err := concurrentExecutor.EnqueueTask(task)
		test.Nil(err)
	}
	concurrentExecutor.StartExecution(context.Background())
	resp := concurrentExecutor.WaitForCompletionAndGetResponse()

	// Assertions
	test.Len(resp, numberOfTestingTasks)
	for i := 1; i <= numberOfTestingTasks; i++ {
		taskID := getTaskIDForTest(i)
		test.Contains(resp, taskID)
		test.Equal(taskID, resp[taskID].Value)
		test.Nil(resp[taskID].Error)
	}

	concurrentExecutor.Close()
}

func TestConcurrentExecutor_MultipleRun(t *testing.T) {
	t.Parallel()
	test := assert.New(t)

	numberOfTestingTasksFirstRun := 10
	numberOfTestingTasksSecondRun := 5

	// Create a ConcurrentExecutor
	concurrentExecutor := NewConcurrentExecutor(defaultMaxConcurrentTasks, defautMaxTaskQueueSize)
	// Adding tasks for the first run
	for i := 1; i <= numberOfTestingTasksFirstRun; i++ {
		taskID := getTaskIDForTest(i)
		task := Task{
			ID:           taskID,
			ExecutorArgs: []interface{}{taskID, false, nil},
			Executor:     executorForTest,
		}
		err := concurrentExecutor.EnqueueTask(task)
		test.Nil(err)
	}
	// Start execution tasks first times
	concurrentExecutor.StartExecution(context.Background())
	resp := concurrentExecutor.WaitForCompletionAndGetResponse()

	// Assertions for the first run
	test.Len(resp, numberOfTestingTasksFirstRun)
	for i := 1; i <= numberOfTestingTasksFirstRun; i++ {
		taskID := getTaskIDForTest(i)
		test.Contains(resp, taskID)
		test.Equal(taskID, resp[taskID].Value)
		test.Nil(resp[taskID].Error)
	}

	// Adding tasks for the second run
	for i := numberOfTestingTasksFirstRun + 1; i <= numberOfTestingTasksFirstRun+numberOfTestingTasksSecondRun; i++ {
		taskID := getTaskIDForTest(i)
		task := Task{
			ID:           taskID,
			ExecutorArgs: []interface{}{taskID, false, nil},
			Executor:     executorForTest,
		}
		err := concurrentExecutor.EnqueueTask(task)
		test.Nil(err)
	}

	// Start execution tasks second times
	concurrentExecutor.StartExecution(context.Background())
	resp = concurrentExecutor.WaitForCompletionAndGetResponse()

	// Assertions for the second run
	test.Len(resp, numberOfTestingTasksSecondRun)
	for i := numberOfTestingTasksFirstRun + 1; i <= numberOfTestingTasksFirstRun+numberOfTestingTasksSecondRun; i++ {
		taskID := getTaskIDForTest(i)
		test.Contains(resp, taskID)
		test.Equal(taskID, resp[taskID].Value)
		test.Nil(resp[taskID].Error)
	}
	concurrentExecutor.Close()
}

func TestConcurrentExecutor_ErrorHandling(t *testing.T) {
	test := assert.New(t)

	// Create a ConcurrentExecutor
	concurrentExecutor := NewConcurrentExecutor(defaultMaxConcurrentTasks, defautMaxTaskQueueSize)
	mapExpectedDummyError := make(map[string]error)
	// Adding tasks
	for i := 1; i <= numberOfTestingTasks; i++ {
		taskID := getTaskIDForTest(i)
		isReturnError := i%2 == 0
		var expectedDummyError error = nil
		if isReturnError {
			expectedDummyError = errors.Errorf("dummy error %d", i)
		}
		mapExpectedDummyError[taskID] = expectedDummyError
		task := Task{
			ID:           taskID,
			ExecutorArgs: []interface{}{taskID, isReturnError, expectedDummyError},
			Executor:     executorForTest,
		}
		err := concurrentExecutor.EnqueueTask(task)
		test.Nil(err)
	}

	// Start execution tasks
	concurrentExecutor.StartExecution(context.Background())
	resp := concurrentExecutor.WaitForCompletionAndGetResponse()

	// Assertions
	test.Len(resp, numberOfTestingTasks)
	for i := 1; i <= numberOfTestingTasks; i++ {
		taskID := getTaskIDForTest(i)
		test.Contains(resp, taskID)

		res := resp[taskID]
		if res.Error != nil {
			test.Equal(mapExpectedDummyError[taskID].Error(), res.Error.Error())
		} else {
			test.Equal(taskID, resp[taskID].Value)
			test.Nil(resp[taskID].Error)
		}
	}

	concurrentExecutor.Close()
}

func TestConcurrentExecutor_PanicHandling(t *testing.T) {
	t.Parallel()
	test := assert.New(t)

	// Create a ConcurrentExecutor
	concurrentExecutor := NewConcurrentExecutor(defaultMaxConcurrentTasks, defautMaxTaskQueueSize)
	mapExpectedPanicError := make(map[string]error)
	// Adding tasks
	for i := 1; i <= numberOfTestingTasks; i++ {
		taskID := getTaskIDForTest(i)
		isPanic := i%2 == 0
		mapExpectedPanicError[taskID] = nil
		expectedDummyPanicMsg := ""
		if isPanic {
			expectedDummyPanicMsg = "dummy panic " + strconv.Itoa(i)
			mapExpectedPanicError[taskID] = errors.Errorf("got panic, recover: %v", expectedDummyPanicMsg)
		}
		task := Task{
			ID:           taskID,
			ExecutorArgs: []interface{}{taskID, isPanic, expectedDummyPanicMsg},
			Executor:     executorForPanicTest,
		}
		err := concurrentExecutor.EnqueueTask(task)
		test.Nil(err)
	}

	// Start execution tasks
	concurrentExecutor.StartExecution(context.Background())
	resp := concurrentExecutor.WaitForCompletionAndGetResponse()

	// Assertions
	test.Len(resp, numberOfTestingTasks)
	for i := 1; i <= numberOfTestingTasks; i++ {
		taskID := getTaskIDForTest(i)
		test.Contains(resp, taskID)

		res := resp[taskID]
		if res.Error != nil {
			test.Equal(mapExpectedPanicError[taskID].Error(), res.Error.Error())
		} else {
			test.Equal(taskID, resp[taskID].Value)
			test.Nil(resp[taskID].Error)
		}
	}

	concurrentExecutor.Close()
}

func TestConcurrentExecutor_EnqueueTask(t *testing.T) {
	t.Run("Panic Enqueue a closed Executor", func(t *testing.T) {
		test := assert.New(t)

		// Panic when enqueue a task into a closed ConcurrentExecutor
		defer func() {
			if r := recover(); r != nil {
				test.Equal(r, ClosedPanicMsg)
			}
		}()

		concurrentExecutor := NewConcurrentExecutor(defaultMaxConcurrentTasks, defautMaxTaskQueueSize)
		concurrentExecutor.Close()
		_ = concurrentExecutor.EnqueueTask(Task{
			ID:           "test",
			ExecutorArgs: nil,
			Executor: func(ctx context.Context, args ...interface{}) (interface{}, error) {
				return "", nil
			},
		})
	})

	t.Run("Enqueue task into a full queue", func(t *testing.T) {
		test := assert.New(t)

		concurrentExecutor := NewConcurrentExecutor(defaultMaxConcurrentTasks, defautMaxTaskQueueSize)

		for i := 1; i <= defautMaxTaskQueueSize; i++ {
			task := Task{
				ID: getTaskIDForTest(i),
				Executor: func(ctx context.Context, args ...interface{}) (interface{}, error) {
					return "", nil
				},
				ExecutorArgs: nil,
			}
			err := concurrentExecutor.EnqueueTask(task)
			test.Nil(err)
		}

		// Enqueue a task into a full queue
		err := concurrentExecutor.EnqueueTask(Task{
			ID:           "test-exceeded-queue-size",
			ExecutorArgs: nil,
			Executor: func(ctx context.Context, args ...interface{}) (interface{}, error) {
				return "", nil
			},
		})

		test.Equal(FullQueueErr, err.Error())
		concurrentExecutor.Close()
	})

}

func TestConcurrentExecutor_StartExecution(t *testing.T) {
	t.Run("Panic StartExecution a closed Executor", func(t *testing.T) {
		test := assert.New(t)

		// Panic when enqueue a task into a closed ConcurrentExecutor
		defer func() {
			if r := recover(); r != nil {
				test.Equal(r, ClosedPanicMsg)
			}
		}()

		concurrentExecutor := NewConcurrentExecutor(defaultMaxConcurrentTasks, defautMaxTaskQueueSize)
		err := concurrentExecutor.EnqueueTask(Task{
			ID:           "test",
			ExecutorArgs: nil,
			Executor: func(ctx context.Context, args ...interface{}) (interface{}, error) {
				return "", nil
			},
		})
		test.Nil(err)
		concurrentExecutor.Close()
		concurrentExecutor.StartExecution(context.Background())
	})
}

func TestConcurrentExecutor_Close(t *testing.T) {
	test := assert.New(t)

	concurrentExecutor := NewConcurrentExecutor(defaultMaxConcurrentTasks, defautMaxTaskQueueSize)
	concurrentExecutor.Close()

	test.Equal(true, concurrentExecutor.closed)
}
