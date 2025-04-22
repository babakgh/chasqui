package worker

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/user/chasqui/pkg/logger"
)

// Dispatcher manages the submission of tasks to the worker pool.
// It provides methods to batch process tasks and handle backpressure.
type Dispatcher struct {
	pool      *Pool
	log       *logger.Logger
	batchSize int
	isRunning atomic.Bool
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// DispatcherConfig holds configuration options for the task dispatcher.
type DispatcherConfig struct {
	// Pool is the worker pool to dispatch tasks to
	Pool *Pool
	// Logger is the logger instance to use
	Logger *logger.Logger
	// BatchSize is the maximum number of tasks to process in a batch
	BatchSize int
}

// NewDispatcher creates a new task dispatcher with the given configuration.
func NewDispatcher(parentCtx context.Context, config DispatcherConfig) (*Dispatcher, error) {
	if config.Pool == nil {
		return nil, fmt.Errorf("worker pool cannot be nil")
	}
	if config.Logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	if config.BatchSize <= 0 {
		return nil, fmt.Errorf("batch size must be greater than 0")
	}

	ctx, cancel := context.WithCancel(parentCtx)

	return &Dispatcher{
		pool:      config.Pool,
		log:       config.Logger,
		batchSize: config.BatchSize,
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// DispatchBatch submits a batch of tasks to the worker pool.
// It returns the number of tasks successfully submitted and any error that occurred.
func (d *Dispatcher) DispatchBatch(tasks []Work) (int, error) {
	if !d.isRunning.Load() {
		return 0, fmt.Errorf("dispatcher is not running")
	}

	submitted := 0
	for _, task := range tasks {
		if err := d.pool.Submit(task); err != nil {
			d.log.Warn("Failed to submit task %s (ID: %s): %v", task.Name(), task.ID(), err)
			return submitted, err
		}
		submitted++
	}

	d.log.Info("Dispatched batch of %d tasks", submitted)
	return submitted, nil
}

// ProcessChannel starts processing tasks from the given channel and submitting them to the worker pool.
// It returns immediately and processes tasks in the background.
func (d *Dispatcher) ProcessChannel(taskChan <-chan Work) error {
	if !d.isRunning.Load() {
		return fmt.Errorf("dispatcher is not running")
	}

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		d.log.Info("Starting channel processor")

		batch := make([]Work, 0, d.batchSize)

		for {
			select {
			case <-d.ctx.Done():
				d.log.Info("Channel processor shutting down")
				// Submit any remaining tasks
				if len(batch) > 0 {
					d.log.Info("Submitting final batch of %d tasks", len(batch))
					d.DispatchBatch(batch)
				}
				return
			case task, ok := <-taskChan:
				if !ok {
					// Channel closed
					d.log.Info("Task channel closed")
					// Submit any remaining tasks
					if len(batch) > 0 {
						d.log.Info("Submitting final batch of %d tasks", len(batch))
						d.DispatchBatch(batch)
					}
					return
				}

				batch = append(batch, task)

				// When batch reaches capacity, submit it
				if len(batch) >= d.batchSize {
					d.log.Debug("Batch full, submitting %d tasks", len(batch))
					d.DispatchBatch(batch)
					batch = make([]Work, 0, d.batchSize)
				}
			}
		}
	}()

	return nil
}

// Start initializes and starts the dispatcher.
func (d *Dispatcher) Start() error {
	if d.isRunning.Load() {
		return fmt.Errorf("dispatcher is already running")
	}

	d.log.Info("Starting task dispatcher")
	d.isRunning.Store(true)
	return nil
}

// Shutdown gracefully stops the dispatcher.
// It waits for all background processing to complete.
func (d *Dispatcher) Shutdown(timeout context.Context) error {
	if !d.isRunning.Load() {
		return fmt.Errorf("dispatcher is not running")
	}

	d.log.Info("Initiating graceful shutdown of dispatcher")
	d.isRunning.Store(false)
	d.cancel() // Signal processors to stop

	// Wait for all processors to complete with timeout
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		d.log.Info("Dispatcher shutdown completed successfully")
		return nil
	case <-timeout.Done():
		return fmt.Errorf("timeout waiting for dispatcher to finish: %v", timeout.Err())
	}
}
