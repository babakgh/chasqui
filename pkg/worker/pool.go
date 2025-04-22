package worker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/user/chasqui/pkg/logger"
)

// Pool represents a worker pool that handles the concurrent execution of Work tasks.
type Pool struct {
	workers     int
	taskQueue   chan Work
	log         *logger.Logger
	workerWg    sync.WaitGroup
	shutdownCtx context.Context
	cancelFunc  context.CancelFunc
	isRunning   atomic.Bool
	metrics     PoolMetrics
}

// PoolConfig holds configuration options for the worker pool.
type PoolConfig struct {
	// Workers is the number of concurrent workers to run
	Workers int
	// QueueSize is the buffer size of the task queue
	QueueSize int
	// Logger is the logger instance to use
	Logger *logger.Logger
}

// PoolMetrics holds metrics for the worker pool.
type PoolMetrics struct {
	TasksQueued    atomic.Int64
	TasksCompleted atomic.Int64
	TasksFailed    atomic.Int64
	TasksProcessed atomic.Int64
}

// NewPool creates a new worker pool with the given configuration.
func NewPool(parentCtx context.Context, config PoolConfig) (*Pool, error) {
	if config.Workers <= 0 {
		return nil, fmt.Errorf("number of workers must be greater than 0")
	}
	if config.QueueSize <= 0 {
		return nil, fmt.Errorf("queue size must be greater than 0")
	}
	if config.Logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	ctx, cancel := context.WithCancel(parentCtx)

	pool := &Pool{
		workers:     config.Workers,
		taskQueue:   make(chan Work, config.QueueSize),
		log:         config.Logger,
		shutdownCtx: ctx,
		cancelFunc:  cancel,
	}

	// Initialize all metric counters to zero
	pool.metrics = PoolMetrics{}
	pool.metrics.TasksQueued.Store(0)
	pool.metrics.TasksCompleted.Store(0)
	pool.metrics.TasksFailed.Store(0)
	pool.metrics.TasksProcessed.Store(0)

	pool.isRunning.Store(false)

	return pool, nil
}

// Start launches the worker goroutines and begins processing tasks.
func (p *Pool) Start() error {
	if p.isRunning.Load() {
		return fmt.Errorf("worker pool is already running")
	}

	p.log.Info("Starting worker pool with %d workers", p.workers)
	p.isRunning.Store(true)

	// Start the workers
	for i := 0; i < p.workers; i++ {
		p.workerWg.Add(1)
		workerId := i
		go p.runWorker(workerId)
	}

	return nil
}

// Submit adds a task to the queue to be executed by a worker.
// It will block if the queue is full.
func (p *Pool) Submit(task Work) error {
	if !p.isRunning.Load() {
		return fmt.Errorf("worker pool is not running")
	}

	select {
	case p.taskQueue <- task:
		p.metrics.TasksQueued.Add(1)
		p.log.Debug("Task queued: %s (ID: %s)", task.Name(), task.ID())
		return nil
	case <-p.shutdownCtx.Done():
		return fmt.Errorf("worker pool is shutting down")
	}
}

// TrySubmit attempts to add a task to the queue without blocking.
// It returns an error if the queue is full or the pool is not running.
func (p *Pool) TrySubmit(task Work) error {
	if !p.isRunning.Load() {
		return fmt.Errorf("worker pool is not running")
	}

	select {
	case p.taskQueue <- task:
		p.metrics.TasksQueued.Add(1)
		p.log.Debug("Task queued: %s (ID: %s)", task.Name(), task.ID())
		return nil
	default:
		return fmt.Errorf("task queue is full")
	case <-p.shutdownCtx.Done():
		return fmt.Errorf("worker pool is shutting down")
	}
}

// Shutdown gracefully shuts down the worker pool.
// It stops accepting new tasks and waits for all workers to finish their current tasks.
func (p *Pool) Shutdown(timeout context.Context) error {
	if !p.isRunning.Load() {
		return fmt.Errorf("worker pool is not running")
	}

	p.log.Info("Initiating graceful shutdown of worker pool")
	p.isRunning.Store(false)
	p.cancelFunc() // Signal workers to stop

	// Wait for all workers to complete with timeout
	done := make(chan struct{})
	go func() {
		p.workerWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.log.Info("Worker pool shutdown completed successfully")
		return nil
	case <-timeout.Done():
		return fmt.Errorf("timeout waiting for workers to finish: %v", timeout.Err())
	}
}

// runWorker is the main worker loop that processes tasks from the queue.
func (p *Pool) runWorker(id int) {
	defer p.workerWg.Done()

	p.log.Debug("Worker %d started", id)

	for {
		select {
		case <-p.shutdownCtx.Done():
			p.log.Debug("Worker %d shutting down", id)
			return
		case task, ok := <-p.taskQueue:
			if !ok {
				p.log.Debug("Worker %d: task queue was closed", id)
				return
			}

			p.executeTask(id, task)
		}
	}
}

// executeTask handles the execution of a single task, including error handling and metrics.
func (p *Pool) executeTask(workerId int, task Work) {
	taskCtx, cancel := context.WithCancel(p.shutdownCtx)
	defer cancel()

	p.log.Debug("Worker %d executing task: %s (ID: %s)", workerId, task.Name(), task.ID())

	err := task.Execute(taskCtx)
	p.metrics.TasksProcessed.Add(1)

	if err != nil {
		p.metrics.TasksFailed.Add(1)
		p.log.Error("Worker %d: task %s (ID: %s) failed: %v", workerId, task.Name(), task.ID(), err)
	} else {
		p.metrics.TasksCompleted.Add(1)
		p.log.Debug("Worker %d: task %s (ID: %s) completed successfully", workerId, task.Name(), task.ID())
	}
}

// GetMetrics returns the current metrics for the worker pool.
func (p *Pool) GetMetrics() PoolMetrics {
	return p.metrics
}
