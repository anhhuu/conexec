package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/anhhuu/conexec"
)

func main1() {
	maxTasks := 20
	concurrentExecutor := conexec.NewConcurrentExecutorBuilder().
		WithMaxTaskQueueSize(maxTasks).
		Build()
	defer concurrentExecutor.Close()

	// Adding tasks
	fmt.Printf("[INFO] START ENQUEUE:\n")
	for i := 1; i <= maxTasks; i++ {
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
				randNum := r1.Intn(50)
				time.Sleep(time.Duration(randNum) * time.Millisecond)
				fmt.Printf("Done task: %s, time elapsed: %s\n", taskID, time.Since(startTime))
				var err error = nil
				if randNum < 25 {
					err = fmt.Errorf("dummy error %d, of task: %s", randNum, taskID)
				}

				return "Response from " + taskID, err
			},
			ExecutorArgs: []interface{}{taskID},
		}

		fmt.Printf("Add task: %s into queue\n", taskID)
		err := concurrentExecutor.EnqueueTask(task)
		if err != nil {
			fmt.Printf("Add task got error: %v", err)
			break
		}
	}

	fmt.Printf("\n[INFO] START EXECUTE:\n")
	concurrentExecutor.StartExecution(context.Background())
	resp := concurrentExecutor.WaitForCompletionAndGetResponse()

	fmt.Printf("\n[INFO] SHOW RESPONSE:\n")
	for taskID, res := range resp {
		fmt.Printf("response of [%s], value is [%v], error is: [%v]\n", taskID, res.Value.(string), res.Error)
	}
}
