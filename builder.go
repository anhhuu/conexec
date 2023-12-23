package conexec

// ConcurrentExecutorBuilder is a builder for creating ConcurrentExecutor instances.
type ConcurrentExecutorBuilder struct {
	maxTaskQueueSize   int
	maxConcurrentTasks int
}

/*
NewConcurrentExecutorBuilder creates a new instance of ConcurrentExecutorBuilder with default values.

Default value:
  - MaxTaskQueueSize: 50
  - MaxConcurrentTasks: 5
*/
func NewConcurrentExecutorBuilder() *ConcurrentExecutorBuilder {
	return &ConcurrentExecutorBuilder{
		maxTaskQueueSize:   defautMaxTaskQueueSize,    // Default value
		maxConcurrentTasks: defaultMaxConcurrentTasks, // Default value
	}
}

/*
WithMaxTaskQueueSize sets the maximum task queue size.

Default: 50
*/
func (builder *ConcurrentExecutorBuilder) WithMaxTaskQueueSize(size int) *ConcurrentExecutorBuilder {
	builder.maxTaskQueueSize = size
	return builder
}

/*
WithMaxConcurrentTasks sets the maximum number of concurrent tasks.

Default: 5
*/
func (builder *ConcurrentExecutorBuilder) WithMaxConcurrentTasks(count int) *ConcurrentExecutorBuilder {
	builder.maxConcurrentTasks = count
	return builder
}

// Build creates a new instance of ConcurrentExecutor based on the builder's configuration.
func (builder *ConcurrentExecutorBuilder) Build() *ConcurrentExecutor {
	return &ConcurrentExecutor{
		maxTaskQueueSize:      builder.maxTaskQueueSize,
		maxConcurrentTasks:    builder.maxConcurrentTasks,
		taskQueueChan:         make(chan Task, defautMaxTaskQueueSize),
		responseChan:          make(chan *TaskResponse, defautMaxTaskQueueSize),
		isTaskQueueChanClosed: false,
		closed:                false,
	}
}
