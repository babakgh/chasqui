package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/user/chasqui/examples"
	"github.com/user/chasqui/pkg/logger"
	"github.com/user/chasqui/pkg/worker"
)

// Configuration for the application
type Config struct {
	WorkerCount  int
	QueueSize    int
	BatchSize    int
	TaskCount    int
	ShutdownWait time.Duration
}

func main() {
	// Initialize logger
	log := logger.New(logger.INFO)
	log.Info("Starting Chasqui - Concurrent Task Processing System")

	// Define configuration
	config := Config{
		WorkerCount:  5,
		QueueSize:    100,
		BatchSize:    10,
		TaskCount:    50,
		ShutdownWait: 10 * time.Second,
	}

	// Create a context that will be canceled on SIGINT or SIGTERM
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Set up signal handling
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signalCh
		log.Info("Received signal: %v", sig)
		cancel()
	}()

	// Create and start the worker pool
	pool, err := worker.NewPool(ctx, worker.PoolConfig{
		Workers:   config.WorkerCount,
		QueueSize: config.QueueSize,
		Logger:    log,
	})
	if err != nil {
		log.Fatal("Failed to create worker pool: %v", err)
	}

	if err := pool.Start(); err != nil {
		log.Fatal("Failed to start worker pool: %v", err)
	}

	// Create and start the dispatcher
	dispatcher, err := worker.NewDispatcher(ctx, worker.DispatcherConfig{
		Pool:      pool,
		Logger:    log,
		BatchSize: config.BatchSize,
	})
	if err != nil {
		log.Fatal("Failed to create dispatcher: %v", err)
	}

	if err := dispatcher.Start(); err != nil {
		log.Fatal("Failed to start dispatcher: %v", err)
	}

	// Generate tasks
	taskGenerator := examples.NewTaskGenerator()
	taskChan := taskGenerator.Generate(ctx, config.TaskCount)

	// Process tasks from the channel
	if err := dispatcher.ProcessChannel(taskChan); err != nil {
		log.Fatal("Failed to process task channel: %v", err)
	}

	// Wait for completion or interruption
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		<-ctx.Done()
		log.Info("Initiating graceful shutdown")

		// Create a timeout context for shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), config.ShutdownWait)
		defer shutdownCancel()

		// Shutdown dispatcher first, then worker pool
		if err := dispatcher.Shutdown(shutdownCtx); err != nil {
			log.Error("Error shutting down dispatcher: %v", err)
		}

		if err := pool.Shutdown(shutdownCtx); err != nil {
			log.Error("Error shutting down worker pool: %v", err)
		}

		log.Info("Shutdown complete")
	}()

	// Create some manual tasks for demonstration
	log.Info("Creating and submitting manual task examples")

	if err := pool.Submit(examples.NewSleepTask(2 * time.Second)); err != nil {
		log.Error("Failed to submit manual task: %v", err)
	}

	if err := pool.Submit(examples.NewProcessingTask("ManualProcessingTask", 5, 3*time.Second)); err != nil {
		log.Error("Failed to submit manual task: %v", err)
	}

	if err := pool.Submit(examples.NewFailingTask(0.8, 500*time.Millisecond)); err != nil {
		log.Error("Failed to submit manual task: %v", err)
	}

	// Wait for shutdown to complete
	wg.Wait()

	// Show final metrics
	metrics := pool.GetMetrics()
	fmt.Println("\nFinal Metrics:")
	fmt.Printf("Tasks Queued:    %d\n", metrics.TasksQueued.Load())
	fmt.Printf("Tasks Processed: %d\n", metrics.TasksProcessed.Load())
	fmt.Printf("Tasks Completed: %d\n", metrics.TasksCompleted.Load())
	fmt.Printf("Tasks Failed:    %d\n", metrics.TasksFailed.Load())

	log.Info("Chasqui has exited")
}
