package hephaestus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/user/chasqui/pkg/logger"
)

// Work represents a task that can be executed by the worker pool.
type Work interface {
	// Execute performs the task work. It accepts a context for cancellation.
	Execute(ctx context.Context) error
	// ID returns a unique identifier for the task.
	ID() string
	// Name returns a human-readable name for the task.
	Name() string
}

// PoolMetrics contains metrics about the worker pool.
type PoolMetrics struct {
	// QueueSize is the current number of tasks in the queue
	QueueSize int
	// ActiveWorkers is the number of workers currently processing tasks
	ActiveWorkers int
	// CompletedTasks is the number of tasks completed successfully
	CompletedTasks uint64
	// FailedTasks is the number of tasks that failed
	FailedTasks uint64
	// TotalTasks is the total number of tasks processed
	TotalTasks uint64
}

// PoolConfig holds configuration options for the worker pool.
type PoolConfig struct {
	// Workers is the number of concurrent workers to run
	Workers int
	// QueueSize is the buffer size of the task queue
	QueueSize int
	// Logger is used to log pool operations
	Logger *logger.Logger
}

// Pool manages concurrent execution of tasks.
type Pool struct {
	workers     int
	taskQueue   chan Work
	log         *logger.Logger
	workerWg    sync.WaitGroup
	shutdownCtx context.Context
	cancelFunc  context.CancelFunc
	isRunning   bool
	metrics     struct {
		mu             sync.RWMutex
		activeWorkers  int
		completedTasks uint64
		failedTasks    uint64
		totalTasks     uint64
	}
}

// NewPool creates a new worker pool with the given configuration.
func NewPool(ctx context.Context, config PoolConfig) (*Pool, error) {
	if config.Workers <= 0 {
		return nil, fmt.Errorf("number of workers must be greater than 0")
	}
	if config.QueueSize <= 0 {
		return nil, fmt.Errorf("queue size must be greater than 0")
	}
	if config.Logger == nil {
		return nil, fmt.Errorf("logger must not be nil")
	}

	shutdownCtx, cancelFunc := context.WithCancel(ctx)

	return &Pool{
		workers:     config.Workers,
		taskQueue:   make(chan Work, config.QueueSize),
		log:         config.Logger,
		shutdownCtx: shutdownCtx,
		cancelFunc:  cancelFunc,
	}, nil
}

// Start initiates the worker pool and starts processing tasks.
func (p *Pool) Start() error {
	if p.isRunning {
		return fmt.Errorf("pool is already running")
	}

	p.log.Info("Starting worker pool with %d workers", p.workers)
	p.isRunning = true

	// Start the worker goroutines
	for i := 0; i < p.workers; i++ {
		workerID := i + 1
		p.workerWg.Add(1)
		go p.runWorker(workerID)
	}

	return nil
}

// Submit adds a task to the worker pool.
// It will block if the queue is full.
func (p *Pool) Submit(task Work) error {
	if !p.isRunning {
		return fmt.Errorf("pool is not running")
	}

	select {
	case <-p.shutdownCtx.Done():
		return fmt.Errorf("pool is shutting down")
	case p.taskQueue <- task:
		return nil
	}
}

// TrySubmit attempts to add a task to the queue without blocking.
func (p *Pool) TrySubmit(task Work) error {
	if !p.isRunning {
		return fmt.Errorf("pool is not running")
	}

	select {
	case <-p.shutdownCtx.Done():
		return fmt.Errorf("pool is shutting down")
	case p.taskQueue <- task:
		return nil
	default:
		return fmt.Errorf("queue is full")
	}
}

// Shutdown gracefully shuts down the worker pool.
// It stops accepting new tasks and waits for all workers to finish.
func (p *Pool) Shutdown(ctx context.Context) error {
	if !p.isRunning {
		return nil
	}

	p.log.Info("Shutting down worker pool")
	p.isRunning = false
	p.cancelFunc() // Signal all operations to cancel

	// Close the task queue to stop accepting new tasks
	close(p.taskQueue)

	// Wait for all workers to finish with a timeout
	done := make(chan struct{})
	go func() {
		p.workerWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.log.Info("Worker pool shut down successfully")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("shutdown timed out: %w", ctx.Err())
	}
}

// runWorker processes tasks from the queue.
func (p *Pool) runWorker(id int) {
	defer p.workerWg.Done()
	p.log.Debug("Worker %d started", id)

	for task := range p.taskQueue {
		// Track active workers
		p.metrics.mu.Lock()
		p.metrics.activeWorkers++
		p.metrics.mu.Unlock()

		// Process the task
		p.executeTask(task)

		// Decrement active workers count
		p.metrics.mu.Lock()
		p.metrics.activeWorkers--
		p.metrics.mu.Unlock()
	}

	p.log.Debug("Worker %d stopped", id)
}

// executeTask runs a single task with proper error handling and metrics.
func (p *Pool) executeTask(task Work) {
	taskID := task.ID()
	taskName := task.Name()
	startTime := time.Now()

	p.log.Debug("Started task '%s' (%s)", taskName, taskID)

	// Create a context for this task that will be canceled if the pool is shutting down
	taskCtx, cancel := context.WithCancel(p.shutdownCtx)
	defer cancel()

	// Execute the task
	err := task.Execute(taskCtx)

	// Update metrics
	p.metrics.mu.Lock()
	p.metrics.totalTasks++
	if err != nil {
		p.metrics.failedTasks++
		p.log.Error("Task '%s' (%s) failed after %v: %v",
			taskName, taskID, time.Since(startTime), err)
	} else {
		p.metrics.completedTasks++
		p.log.Debug("Task '%s' (%s) completed in %v",
			taskName, taskID, time.Since(startTime))
	}
	p.metrics.mu.Unlock()
}

// GetMetrics returns the current metrics for the worker pool.
func (p *Pool) GetMetrics() PoolMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	return PoolMetrics{
		QueueSize:      len(p.taskQueue),
		ActiveWorkers:  p.metrics.activeWorkers,
		CompletedTasks: p.metrics.completedTasks,
		FailedTasks:    p.metrics.failedTasks,
		TotalTasks:     p.metrics.totalTasks,
	}
}
