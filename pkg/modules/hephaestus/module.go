package hephaestus

import (
	"context"
	"fmt"

	"github.com/user/chasqui/pkg/logger"
	"github.com/user/chasqui/pkg/module"
)

// Module represents the Hephaestus module that provides worker pool functionality.
type Module struct {
	ctx        context.Context
	cancel     context.CancelFunc
	log        *logger.Logger
	pool       *Pool
	dispatcher *Dispatcher
	config     Config
	taskChan   chan Work
}

// Config holds configuration options for the Hephaestus module.
type Config struct {
	// Workers is the number of concurrent workers to run
	Workers int `yaml:"workers"`
	// QueueSize is the buffer size of the task queue
	QueueSize int `yaml:"queueSize"`
	// BatchSize is the maximum number of tasks to process in a batch
	BatchSize int `yaml:"batchSize"`
	// ShutdownTimeout is the maximum time to wait for the pool to shutdown gracefully
	ShutdownTimeout module.Duration `yaml:"shutdownTimeout"`
}

// NewModule creates a new instance of the Hephaestus module.
func NewModule(ctx context.Context, log *logger.Logger, config Config) (module.Module, error) {
	if config.Workers <= 0 {
		return nil, fmt.Errorf("number of workers must be greater than 0")
	}
	if config.QueueSize <= 0 {
		return nil, fmt.Errorf("queue size must be greater than 0")
	}
	if config.BatchSize <= 0 {
		return nil, fmt.Errorf("batch size must be greater than 0")
	}

	ctx, cancel := context.WithCancel(ctx)

	return &Module{
		ctx:      ctx,
		cancel:   cancel,
		log:      log,
		config:   config,
		taskChan: make(chan Work, config.QueueSize),
	}, nil
}

// Name returns the name of the module.
func (m *Module) Name() string {
	return "hephaestus"
}

// Init initializes the module with the provided config.
// Since we're initializing in NewModule, this just validates the config.
func (m *Module) Init(cfg any) error {
	// Configuration is already validated in NewModule, so nothing to do here
	m.log.Debug("Initializing Hephaestus module")
	return nil
}

// Start initializes the worker pool and starts it.
func (m *Module) Start() error {
	m.log.Info("Starting Hephaestus module")

	// Initialize the worker pool
	var err error
	m.pool, err = NewPool(m.ctx, PoolConfig{
		Workers:   m.config.Workers,
		QueueSize: m.config.QueueSize,
		Logger:    m.log,
	})
	if err != nil {
		return fmt.Errorf("failed to create worker pool: %w", err)
	}

	// Start the worker pool
	if err := m.pool.Start(); err != nil {
		return fmt.Errorf("failed to start worker pool: %w", err)
	}

	// Initialize the dispatcher
	m.dispatcher, err = NewDispatcher(m.ctx, DispatcherConfig{
		Pool:      m.pool,
		Logger:    m.log,
		BatchSize: m.config.BatchSize,
	})
	if err != nil {
		return fmt.Errorf("failed to create task dispatcher: %w", err)
	}

	// Start the dispatcher
	if err := m.dispatcher.Start(); err != nil {
		return fmt.Errorf("failed to start task dispatcher: %w", err)
	}

	// Start processing tasks from the input channel
	if err := m.dispatcher.ProcessChannel(m.taskChan); err != nil {
		return fmt.Errorf("failed to start processing task channel: %w", err)
	}

	m.log.Info("Hephaestus module started successfully")

	// Block until context is done
	<-m.ctx.Done()
	return nil
}

// Stop gracefully shuts down the worker pool.
func (m *Module) Stop(ctx context.Context) error {
	m.log.Info("Stopping Hephaestus module")

	// Close the task channel to stop accepting new tasks
	close(m.taskChan)

	// Shutdown the dispatcher
	if m.dispatcher != nil {
		if err := m.dispatcher.Shutdown(ctx); err != nil {
			m.log.Warn("Error shutting down task dispatcher: %v", err)
		}
	}

	// Shutdown the worker pool
	if m.pool != nil {
		if err := m.pool.Shutdown(ctx); err != nil {
			m.log.Warn("Error shutting down worker pool: %v", err)
			return fmt.Errorf("error shutting down worker pool: %w", err)
		}
	}

	m.cancel()
	m.log.Info("Hephaestus module stopped successfully")
	return nil
}

// Dependencies returns other module names this module depends on.
// Hephaestus doesn't depend on any other modules.
func (m *Module) Dependencies() []string {
	return []string{}
}

// TaskChannel returns the channel that tasks can be submitted to.
// This is the public API for the module.
func (m *Module) TaskChannel() chan<- Work {
	return m.taskChan
}

// Submit adds a task to the worker pool.
// It will block if the queue is full.
func (m *Module) Submit(task Work) error {
	select {
	case <-m.ctx.Done():
		return fmt.Errorf("module is shutting down")
	case m.taskChan <- task:
		return nil
	}
}

// TrySubmit attempts to add a task to the queue without blocking.
func (m *Module) TrySubmit(task Work) error {
	select {
	case <-m.ctx.Done():
		return fmt.Errorf("module is shutting down")
	case m.taskChan <- task:
		return nil
	default:
		return fmt.Errorf("queue is full")
	}
}

// GetMetrics returns the current metrics for the worker pool.
func (m *Module) GetMetrics() PoolMetrics {
	if m.pool == nil {
		return PoolMetrics{}
	}
	return m.pool.GetMetrics()
}
