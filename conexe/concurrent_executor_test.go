package conexe

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConcurrentExecutor(t *testing.T) {
	t.Parallel()

	t.Run("happy case with 1 time run", func(t *testing.T) {
		// Create a ConcurrentExecutor
		concurrentExecutor := NewConcurrentExecutor(10, 5)

		// Adding tasks
		for i := 1; i <= 5; i++ {
			taskID := fmt.Sprintf("Task_%d", i)
			task := Task{
				ID: taskID,
				Executor: func(ctx context.Context, args ...interface{}) (interface{}, error) {
					taskID, ok := args[0].(string)
					if !ok {
						return "", fmt.Errorf("parse error")
					}

					// Simulating task execution
					time.Sleep(100 * time.Millisecond)
					return taskID, nil
				},
				ExecutorArgs: []interface{}{taskID},
			}
			err := concurrentExecutor.EnqueueTask(task)
			assert.Nil(t, err)
		}
		concurrentExecutor.StartExecution(context.Background())
		resp := concurrentExecutor.WaitForCompletionAndGetResponse()

		// Assertions
		assert.Len(t, resp, 5)
		for i := 1; i <= 5; i++ {
			taskID := fmt.Sprintf("Task_%d", i)
			assert.Contains(t, resp, taskID)
			assert.Equal(t, fmt.Sprintf("Task_%d", i), resp[taskID].Value)
			assert.Nil(t, resp[taskID].Error)
		}

		concurrentExecutor.Close()
	})

	t.Run("happy case with 2 times run", func(t *testing.T) {
		// Create a ConcurrentExecutor
		concurrentExecutor := NewConcurrentExecutor(3, 10)
		// Adding tasks
		for i := 1; i <= 5; i++ {
			taskID := fmt.Sprintf("Task_%d", i)
			task := Task{
				ID: taskID,
				Executor: func(ctx context.Context, args ...interface{}) (interface{}, error) {
					taskID, ok := args[0].(string)
					if !ok {
						return "", fmt.Errorf("parse error")
					}

					// Simulating task execution
					time.Sleep(100 * time.Millisecond)
					return taskID, nil
				},
				ExecutorArgs: []interface{}{taskID},
			}
			err := concurrentExecutor.EnqueueTask(task)
			assert.Nil(t, err)
		}
		// First run
		concurrentExecutor.StartExecution(context.Background())
		resp := concurrentExecutor.WaitForCompletionAndGetResponse()

		// Assertions
		assert.Len(t, resp, 5)
		for i := 1; i <= 5; i++ {
			taskID := fmt.Sprintf("Task_%d", i)
			assert.Contains(t, resp, taskID)
			assert.Equal(t, fmt.Sprintf("Task_%d", i), resp[taskID].Value)
			assert.Nil(t, resp[taskID].Error)
		}

		// Adding tasks for the second run
		for i := 6; i <= 10; i++ {
			taskID := fmt.Sprintf("Task_%d", i)
			task := Task{
				ID: taskID,
				Executor: func(ctx context.Context, args ...interface{}) (interface{}, error) {
					taskID, ok := args[0].(string)
					if !ok {
						return "", fmt.Errorf("parse error")
					}

					// Simulating task execution
					time.Sleep(100 * time.Millisecond)
					return taskID, nil
				},
				ExecutorArgs: []interface{}{taskID},
			}
			err := concurrentExecutor.EnqueueTask(task)
			assert.Nil(t, err)
		}
		// Second run
		concurrentExecutor.StartExecution(context.Background())
		resp = concurrentExecutor.WaitForCompletionAndGetResponse()

		// Assertions for the second run
		assert.Len(t, resp, 5)
		for i := 6; i <= 10; i++ {
			taskID := fmt.Sprintf("Task_%d", i)
			assert.Contains(t, resp, taskID)
			assert.Equal(t, fmt.Sprintf("Task_%d", i), resp[taskID].Value)
			assert.Nil(t, resp[taskID].Error)
		}
		concurrentExecutor.Close()
	})

	t.Run("error handling", func(t *testing.T) {
		// Create a ConcurrentExecutor
		concurrentExecutor := NewConcurrentExecutor(10, 5)
		expectedError := make(map[string]error)
		// Adding tasks
		for i := 1; i <= 5; i++ {
			taskID := fmt.Sprintf("Task_%d", i)
			expectedError[taskID] = fmt.Errorf("dummy error %d", i)
			task := Task{
				ID: taskID,
				Executor: func(ctx context.Context, args ...interface{}) (interface{}, error) {
					taskID, ok := args[0].(string)
					if !ok {
						return "", fmt.Errorf("parse task_id error")
					}
					index, ok := args[1].(int)
					if !ok {
						return "", fmt.Errorf("parse index error")
					}

					// Simulating task execution
					time.Sleep(100 * time.Millisecond)
					if index%2 == 0 {
						return "", expectedError[taskID]
					}

					return taskID, nil
				},
				ExecutorArgs: []interface{}{taskID, i},
			}
			err := concurrentExecutor.EnqueueTask(task)
			assert.Nil(t, err)
		}
		concurrentExecutor.StartExecution(context.Background())
		resp := concurrentExecutor.WaitForCompletionAndGetResponse()

		// Assertions
		assert.Len(t, resp, 5)
		for i := 1; i <= 5; i++ {
			taskID := fmt.Sprintf("Task_%d", i)
			assert.Contains(t, resp, taskID)

			res := resp[taskID]
			if res.Error != nil {
				assert.Equal(t, expectedError[taskID].Error(), res.Error.Error())
			} else {
				assert.Equal(t, taskID, resp[taskID].Value)
				assert.Nil(t, resp[taskID].Error)
			}
		}

		concurrentExecutor.Close()
	})
}
