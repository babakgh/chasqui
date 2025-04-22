# Chasqui: Concurrent Background Task Processing in Go

Chasqui is a Go library for executing multiple background tasks concurrently using a worker pool pattern. The library provides a simple, efficient way to process tasks asynchronously with proper error handling, logging, and graceful shutdown capabilities.

## Features

- **Worker Pool**: Manages a configurable number of goroutines to process tasks concurrently
- **Task Interface**: Common `Work` interface for implementing diverse task types
- **Dispatcher**: Batches and schedules tasks efficiently
- **Graceful Shutdown**: Proper handling of in-progress tasks during shutdown
- **Error Handling**: Robust error reporting and recovery
- **Logging**: Comprehensive logging of task lifecycle events
- **Backpressure Handling**: Mechanisms to handle queue overflow
- **Metrics**: Basic task processing metrics

## Installation

```bash
go get github.com/user/chasqui
```

## Quick Start

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/user/chasqui/examples"
	"github.com/user/chasqui/pkg/logger"
	"github.com/user/chasqui/pkg/worker"
)

func main() {
	// Create a logger
	log := logger.New(logger.INFO)

	// Create a worker pool with 5 workers and a queue size of 100
	pool, err := worker.NewPool(worker.PoolConfig{
		Workers:   5,
		QueueSize: 100,
		Logger:    log,
	})
	if err != nil {
		log.Fatal("Failed to create worker pool: %v", err)
	}

	// Start the worker pool
	if err := pool.Start(); err != nil {
		log.Fatal("Failed to start worker pool: %v", err)
	}

	// Create a task
	task := examples.NewSleepTask(2 * time.Second)

	// Submit the task to the pool
	if err := pool.Submit(task); err != nil {
		log.Error("Failed to submit task: %v", err)
	}

	// Wait for signals to gracefully shutdown
	ctx, cancel := context.WithCancel(context.Background())
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalCh
		cancel()
	}()

	// Wait for cancellation
	<-ctx.Done()

	// Gracefully shutdown the pool
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := pool.Shutdown(shutdownCtx); err != nil {
		log.Error("Error shutting down worker pool: %v", err)
	}
}
```

## Creating Custom Tasks

To create a custom task, implement the `Work` interface:

```go
type MyCustomTask struct {
	id   string
	name string
	// Custom task fields
}

// NewMyCustomTask creates a new custom task
func NewMyCustomTask() *MyCustomTask {
	return &MyCustomTask{
		id:   uuid.New().String(),
		name: "MyCustomTask",
	}
}

// Execute implements the worker.Work interface
func (t *MyCustomTask) Execute(ctx context.Context) error {
	// Implement your task logic here
	// Check for ctx.Done() to handle cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Do work
		return nil
	}
}

// ID implements the worker.Work interface
func (t *MyCustomTask) ID() string {
	return t.id
}

// Name implements the worker.Work interface
func (t *MyCustomTask) Name() string {
	return t.name
}
```

## Using the Dispatcher

For more advanced usage, you can use the dispatcher to batch process tasks:

```go
// Create and start a dispatcher
dispatcher, err := worker.NewDispatcher(worker.DispatcherConfig{
	Pool:      pool,
	Logger:    log,
	BatchSize: 10,
})
if err != nil {
	log.Fatal("Failed to create dispatcher: %v", err)
}

if err := dispatcher.Start(); err != nil {
	log.Fatal("Failed to start dispatcher: %v", err)
}

// Create a channel of tasks
taskChan := make(chan worker.Work, 100)

// Process tasks from the channel
if err := dispatcher.ProcessChannel(taskChan); err != nil {
	log.Fatal("Failed to process task channel: %v", err)
}

// Submit a batch of tasks
tasks := []worker.Work{
	examples.NewSleepTask(1 * time.Second),
	examples.NewSleepTask(2 * time.Second),
}
submitted, err := dispatcher.DispatchBatch(tasks)
if err != nil {
	log.Error("Failed to dispatch batch: %v", err)
}
log.Info("Submitted %d tasks", submitted)

// Shutdown the dispatcher
if err := dispatcher.Shutdown(shutdownCtx); err != nil {
	log.Error("Error shutting down dispatcher: %v", err)
}
```

## Handling Backpressure

To handle queue overflow, you can use the `TrySubmit` method instead of `Submit`:

```go
// TrySubmit will return an error immediately if the queue is full
if err := pool.TrySubmit(task); err != nil {
	if err.Error() == "task queue is full" {
		// Handle backpressure
		log.Warn("Task queue is full, implementing backoff strategy")
		// ... backoff logic ...
	} else {
		log.Error("Failed to submit task: %v", err)
	}
}
```

## Configuration

The worker pool and dispatcher can be configured with various options:

```go
// Pool configuration
poolConfig := worker.PoolConfig{
	Workers:   10,               // Number of concurrent workers
	QueueSize: 1000,             // Size of the task queue
	Logger:    logger.New(logger.DEBUG),  // Logger with desired level
}

// Dispatcher configuration
dispatcherConfig := worker.DispatcherConfig{
	Pool:      pool,             // The worker pool to use
	Logger:    log,              // Logger instance
	BatchSize: 50,               // Maximum batch size
}
```

## Metrics

You can retrieve metrics from the worker pool:

```go
metrics := pool.GetMetrics()
fmt.Printf("Tasks Queued:    %d\n", metrics.TasksQueued.Load())
fmt.Printf("Tasks Processed: %d\n", metrics.TasksProcessed.Load())
fmt.Printf("Tasks Completed: %d\n", metrics.TasksCompleted.Load())
fmt.Printf("Tasks Failed:    %d\n", metrics.TasksFailed.Load())
```

## License

MIT 