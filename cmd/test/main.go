package main

import (
	"context"
	"fmt"
	"time"

	"github.com/user/chasqui/examples"
	"github.com/user/chasqui/pkg/logger"
	"github.com/user/chasqui/pkg/worker"
)

func main() {
	// Create a simple test that runs a few tasks and waits for them to complete
	fmt.Println("=== Chasqui Task Completion Test ===")

	// Create a logger with DEBUG level
	log := logger.New(logger.DEBUG)

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a worker pool
	pool, err := worker.NewPool(ctx, worker.PoolConfig{
		Workers:   3,
		QueueSize: 10,
		Logger:    log,
	})

	if err != nil {
		fmt.Printf("Error creating pool: %v\n", err)
		return
	}

	// Start the pool
	if err := pool.Start(); err != nil {
		fmt.Printf("Error starting pool: %v\n", err)
		return
	}

	fmt.Println("Created worker pool with 3 workers")

	// Create and submit tasks
	tasks := []worker.Work{
		examples.NewSleepTask(100*time.Millisecond, log),
		examples.NewProcessingTask("TestProcess", 3, 150*time.Millisecond, log),
		examples.NewFailingTask(0.5, 50*time.Millisecond, log),
	}

	fmt.Println("Submitting 3 test tasks...")

	for _, task := range tasks {
		if err := pool.Submit(task); err != nil {
			fmt.Printf("Error submitting task: %v\n", err)
		} else {
			fmt.Printf("Submitted task: %s (ID: %s)\n", task.Name(), task.ID())
		}
	}

	// Wait for tasks to complete (with a safe timeout)
	fmt.Println("Waiting for tasks to complete...")
	time.Sleep(500 * time.Millisecond)

	// Print metrics
	metrics := pool.GetMetrics()
	fmt.Println("\nTask Metrics:")
	fmt.Printf("  Tasks Queued:    %d\n", metrics.TasksQueued.Load())
	fmt.Printf("  Tasks Processed: %d\n", metrics.TasksProcessed.Load())
	fmt.Printf("  Tasks Completed: %d\n", metrics.TasksCompleted.Load())
	fmt.Printf("  Tasks Failed:    %d\n", metrics.TasksFailed.Load())

	// Shutdown the pool
	fmt.Println("\nShutting down pool...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer shutdownCancel()

	if err := pool.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("Error shutting down pool: %v\n", err)
	}

	fmt.Println("Test complete")
}
