## Run

```
go run ./simple.go
```

## Output:

Example output for 20 tasks in queue, and 5 for concurrent

```
[INFO] START ENQUEUE:
Add task: Task_1 into queue
Add task: Task_2 into queue
Add task: Task_3 into queue
Add task: Task_4 into queue
Add task: Task_5 into queue
Add task: Task_6 into queue
Add task: Task_7 into queue
Add task: Task_8 into queue
Add task: Task_9 into queue
Add task: Task_10 into queue
Add task: Task_11 into queue
Add task: Task_12 into queue
Add task: Task_13 into queue
Add task: Task_14 into queue
Add task: Task_15 into queue
Add task: Task_16 into queue
Add task: Task_17 into queue
Add task: Task_18 into queue
Add task: Task_19 into queue
Add task: Task_20 into queue

[INFO] START EXECUTE:
Start task: Task_2
Start task: Task_3
Start task: Task_1
Start task: Task_4
Start task: Task_5
Done task: Task_3, time elapsed: 9.745209ms
Start task: Task_6
Done task: Task_1, time elapsed: 13.463417ms
Start task: Task_7
Done task: Task_4, time elapsed: 35.096583ms
Start task: Task_8
Done task: Task_7, time elapsed: 29.294166ms
Start task: Task_9
Done task: Task_5, time elapsed: 45.082833ms
Start task: Task_10
Done task: Task_6, time elapsed: 35.34525ms
Done task: Task_2, time elapsed: 45.367ms
Start task: Task_11
Start task: Task_12
Done task: Task_8, time elapsed: 12.351834ms
Start task: Task_13
Done task: Task_12, time elapsed: 5.412291ms
Start task: Task_14
Done task: Task_14, time elapsed: 3.413959ms
Start task: Task_15
Done task: Task_15, time elapsed: 2.2995ms
Start task: Task_16
Done task: Task_13, time elapsed: 11.041791ms
Start task: Task_17
Done task: Task_11, time elapsed: 13.29075ms
Start task: Task_18
Done task: Task_16, time elapsed: 12.070208ms
Start task: Task_19
Done task: Task_9, time elapsed: 32.2655ms
Start task: Task_20
Done task: Task_18, time elapsed: 19.380125ms
Done task: Task_19, time elapsed: 15.812583ms
Done task: Task_10, time elapsed: 46.891458ms
Done task: Task_17, time elapsed: 35.261584ms
Done task: Task_20, time elapsed: 24.124083ms

[INFO] SHOW RESPONSE:
response of [Task_7], value is [Response from Task_7], error is: [<nil>]
response of [Task_6], value is [Response from Task_6], error is: [<nil>]
response of [Task_2], value is [Response from Task_2], error is: [<nil>]
response of [Task_12], value is [Response from Task_12], error is: [dummy error 5, of task: Task_12]
response of [Task_14], value is [Response from Task_14], error is: [dummy error 3, of task: Task_14]
response of [Task_11], value is [Response from Task_11], error is: [dummy error 13, of task: Task_11]
response of [Task_9], value is [Response from Task_9], error is: [<nil>]
response of [Task_1], value is [Response from Task_1], error is: [dummy error 13, of task: Task_1]
response of [Task_16], value is [Response from Task_16], error is: [dummy error 11, of task: Task_16]
response of [Task_18], value is [Response from Task_18], error is: [dummy error 19, of task: Task_18]
response of [Task_8], value is [Response from Task_8], error is: [dummy error 12, of task: Task_8]
response of [Task_10], value is [Response from Task_10], error is: [<nil>]
response of [Task_17], value is [Response from Task_17], error is: [<nil>]
response of [Task_3], value is [Response from Task_3], error is: [dummy error 9, of task: Task_3]
response of [Task_4], value is [Response from Task_4], error is: [<nil>]
response of [Task_5], value is [Response from Task_5], error is: [<nil>]
response of [Task_15], value is [Response from Task_15], error is: [dummy error 2, of task: Task_15]
response of [Task_13], value is [Response from Task_13], error is: [dummy error 11, of task: Task_13]
response of [Task_19], value is [Response from Task_19], error is: [dummy error 15, of task: Task_19]
response of [Task_20], value is [Response from Task_20], error is: [dummy error 24, of task: Task_20]
```