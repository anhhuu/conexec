package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/anhhuu/conexec"
)

func main() {
	concurrentExecutor := conexec.NewConcurrentExecutorBuilder().
		WithMaxTaskQueueSize(15).
		WithMaxConcurrentTasks(5).
		Build()

	// Adding tasks
	for i := 1; i <= 15; i++ {
		taskID := fmt.Sprintf("Task_%d", i)
		task := conexec.Task{
			ID: taskID,
			Executor: func(ctx context.Context, args ...interface{}) (interface{}, error) {
				taskID, ok := args[0].(string)
				if !ok {
					return "", fmt.Errorf("parse error")
				}
				fmt.Printf("Start task: %s\n", taskID)
				// Simulating task execution
				startTime := time.Now()
				s1 := rand.NewSource(time.Now().UnixNano())
				r1 := rand.New(s1)
				time.Sleep(time.Duration(r1.Intn(50)) * time.Millisecond)
				fmt.Printf("Done task: %s, time elapsed: %s\n", taskID, time.Since(startTime))
				return "Response from " + taskID, nil
			},
			ExecutorArgs: []interface{}{taskID},
		}

		err := concurrentExecutor.EnqueueTask(task)
		if err != nil {
			fmt.Printf("Add task got error: %v", err)
			break
		}
	}

	concurrentExecutor.StartExecution(context.Background())
	resp := concurrentExecutor.WaitForCompletionAndGetResponse()

	fmt.Println()
	for taskID, res := range resp {
		fmt.Printf("response of [%s], value is [%v], error is: [%v]\n", taskID, res.Value.(string), res.Error)
	}
}
