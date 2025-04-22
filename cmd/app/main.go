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
	WorkerCount    int
	QueueSize      int
	BatchSize      int
	TaskCount      int
	ShutdownWait   time.Duration
	LogLevel       logger.LogLevel
	SkipBatchTasks bool // Option to skip batch task processing for faster testing
}

// AppContext holds all application components
type AppContext struct {
	config     Config
	log        *logger.Logger
	pool       *worker.Pool
	dispatcher *worker.Dispatcher
	ctx        context.Context
	cancel     context.CancelFunc
}

func main() {
	// Initialize application
	app := initializeApp()

	// Run the application
	if err := runApplication(app); err != nil {
		app.log.Fatal("Application failed: %v", err)
	}

	app.log.Info("Chasqui has exited")
}

// initializeApp creates and initializes the application components
func initializeApp() *AppContext {
	// Create application config
	config := Config{
		WorkerCount:    5,
		QueueSize:      100,
		BatchSize:      10,
		TaskCount:      10,
		ShutdownWait:   10 * time.Second,
		LogLevel:       logger.DEBUG,
		SkipBatchTasks: true, // Test only manual tasks
	}

	// Initialize logger
	log := logger.New(config.LogLevel)
	log.Info("Starting Chasqui - Concurrent Background Task Processing System")

	// Create a context that will be canceled on SIGINT or SIGTERM
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	return &AppContext{
		config: config,
		log:    log,
		ctx:    ctx,
		cancel: cancel,
	}
}

// runApplication runs the main application logic
func runApplication(app *AppContext) error {
	// Set up signal handling for graceful shutdown
	setupSignalHandling(app)

	// Initialize worker pool and dispatcher
	if err := setupWorkerComponents(app); err != nil {
		return err
	}

	// Create and submit some manual task examples
	submitManualTasks(app)

	// Generate and process batch tasks if not skipped
	if !app.config.SkipBatchTasks {
		if err := processBatchTasks(app); err != nil {
			return err
		}
	} else {
		app.log.Info("Skipping batch task processing for testing")
	}

	// Wait for completion or interruption
	waitForCompletion(app)

	// Display final metrics
	displayMetrics(app.pool)

	return nil
}

// setupSignalHandling configures signal handling for graceful shutdown
func setupSignalHandling(app *AppContext) {
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signalCh
		app.log.Info("Received signal: %v", sig)
		app.cancel()
	}()
}

// setupWorkerComponents creates and initializes the worker pool and dispatcher
func setupWorkerComponents(app *AppContext) error {
	var err error

	// Create and start the worker pool
	app.pool, err = worker.NewPool(app.ctx, worker.PoolConfig{
		Workers:   app.config.WorkerCount,
		QueueSize: app.config.QueueSize,
		Logger:    app.log,
	})

	if err != nil {
		return fmt.Errorf("failed to create worker pool: %w", err)
	}

	if err := app.pool.Start(); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	// Create and start the dispatcher
	app.dispatcher, err = worker.NewDispatcher(app.ctx, worker.DispatcherConfig{
		Pool:      app.pool,
		Logger:    app.log,
		BatchSize: app.config.BatchSize,
	})

	if err != nil {
		return fmt.Errorf("failed to create dispatcher: %w", err)
	}

	if err := app.dispatcher.Start(); err != nil {
		return fmt.Errorf("failed to start dispatcher: %w", err)
	}

	return nil
}

// submitManualTasks creates and submits individual task examples
func submitManualTasks(app *AppContext) {
	app.log.Info("Creating and submitting manual task examples")

	// Create a sleep task
	sleepTask := examples.NewSleepTask(200*time.Millisecond, app.log)
	if err := app.pool.Submit(sleepTask); err != nil {
		app.log.Error("Failed to submit sleep task: %v", err)
	}

	// Create a processing task
	processingTask := examples.NewProcessingTask("ManualProcessingTask", 5, 300*time.Millisecond, app.log)
	if err := app.pool.Submit(processingTask); err != nil {
		app.log.Error("Failed to submit processing task: %v", err)
	}

	// Create a failing task
	failingTask := examples.NewFailingTask(0.8, 100*time.Millisecond, app.log)
	if err := app.pool.Submit(failingTask); err != nil {
		app.log.Error("Failed to submit failing task: %v", err)
	}
}

// processBatchTasks generates and processes batch tasks
func processBatchTasks(app *AppContext) error {
	// Create task generator
	taskGenerator := examples.NewTaskGenerator(app.log)

	// Generate tasks
	taskChan := taskGenerator.Generate(app.ctx, app.config.TaskCount)

	// Process tasks from the channel
	if err := app.dispatcher.ProcessChannel(taskChan); err != nil {
		return fmt.Errorf("failed to process task channel: %w", err)
	}

	return nil
}

// waitForCompletion waits for the application to complete or be interrupted
func waitForCompletion(app *AppContext) {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		<-app.ctx.Done()
		app.log.Info("Initiating graceful shutdown")

		// Create a timeout context for shutdown
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), app.config.ShutdownWait)
		defer shutdownCancel()

		// Shutdown dispatcher first, then worker pool
		if err := app.dispatcher.Shutdown(shutdownCtx); err != nil {
			app.log.Error("Error shutting down dispatcher: %v", err)
		}

		if err := app.pool.Shutdown(shutdownCtx); err != nil {
			app.log.Error("Error shutting down worker pool: %v", err)
		}

		app.log.Info("Shutdown complete")
	}()

	// Wait for shutdown to complete
	wg.Wait()
}

// displayMetrics shows the final worker pool metrics
func displayMetrics(pool *worker.Pool) {
	metrics := pool.GetMetrics()

	fmt.Println("\nFinal Metrics:")
	fmt.Printf("Tasks Queued:    %d\n", metrics.TasksQueued.Load())
	fmt.Printf("Tasks Processed: %d\n", metrics.TasksProcessed.Load())
	fmt.Printf("Tasks Completed: %d\n", metrics.TasksCompleted.Load())
	fmt.Printf("Tasks Failed:    %d\n", metrics.TasksFailed.Load())

	completionRate := 0.0
	if metrics.TasksProcessed.Load() > 0 {
		completionRate = float64(metrics.TasksCompleted.Load()) / float64(metrics.TasksProcessed.Load()) * 100
	}

	fmt.Printf("Completion Rate: %.1f%%\n", completionRate)
}
