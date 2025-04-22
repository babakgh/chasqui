package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/user/chasqui/pkg/logger"
)

// MockWork is a simple implementation of the Work interface for testing.
type MockWork struct {
	id        string
	name      string
	execFn    func(ctx context.Context) error
	execCount int
}

func NewMockWork(id, name string, execFn func(ctx context.Context) error) *MockWork {
	return &MockWork{
		id:     id,
		name:   name,
		execFn: execFn,
	}
}

func (m *MockWork) Execute(ctx context.Context) error {
	m.execCount++
	return m.execFn(ctx)
}

func (m *MockWork) ID() string {
	return m.id
}

func (m *MockWork) Name() string {
	return m.name
}

func TestPoolCreation(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logger.New(logger.INFO)

	// Test with invalid configuration
	testCases := []struct {
		name   string
		config PoolConfig
	}{
		{
			name: "zero workers",
			config: PoolConfig{
				Workers:   0,
				QueueSize: 10,
				Logger:    logger.New(logger.INFO),
			},
		},
		{
			name: "zero queue size",
			config: PoolConfig{
				Workers:   5,
				QueueSize: 0,
				Logger:    logger.New(logger.INFO),
			},
		},
		{
			name: "nil logger",
			config: PoolConfig{
				Workers:   5,
				QueueSize: 10,
				Logger:    nil,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewPool(parentCtx, tc.config)
			if err == nil {
				t.Errorf("Expected error for invalid config %s, got nil", tc.name)
			}
		})
	}

	// Test with valid configuration
	pool, err := NewPool(parentCtx, PoolConfig{
		Workers:   5,
		QueueSize: 10,
		Logger:    log,
	})

	if err != nil {
		t.Fatalf("Error creating pool with valid config: %v", err)
	}

	if pool == nil {
		t.Fatal("Pool should not be nil with valid config")
	}
}

func TestPoolStartAndSubmit(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logger.New(logger.INFO)
	pool, err := NewPool(parentCtx, PoolConfig{
		Workers:   2,
		QueueSize: 5,
		Logger:    log,
	})

	if err != nil {
		t.Fatalf("Error creating pool: %v", err)
	}

	// Try submitting before starting
	mockTask := NewMockWork("1", "test", func(ctx context.Context) error {
		return nil
	})

	err = pool.Submit(mockTask)
	if err == nil {
		t.Error("Should not be able to submit task before pool is started")
	}

	// Start the pool
	err = pool.Start()
	if err != nil {
		t.Fatalf("Error starting pool: %v", err)
	}

	// Try starting again
	err = pool.Start()
	if err == nil {
		t.Error("Should not be able to start pool twice")
	}

	// Submit a task
	err = pool.Submit(mockTask)
	if err != nil {
		t.Errorf("Error submitting task: %v", err)
	}

	// Wait a bit for the task to be processed
	time.Sleep(100 * time.Millisecond)

	// Shutdown the pool
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = pool.Shutdown(ctx)
	if err != nil {
		t.Errorf("Error shutting down pool: %v", err)
	}
}

func TestPoolTaskExecution(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logger.New(logger.INFO)
	pool, _ := NewPool(parentCtx, PoolConfig{
		Workers:   3,
		QueueSize: 10,
		Logger:    log,
	})

	pool.Start()

	// Create tasks with different behaviors
	successfulTaskExecuted := false
	successfulTask := NewMockWork("success", "success task", func(ctx context.Context) error {
		successfulTaskExecuted = true
		return nil
	})

	errorTaskExecuted := false
	errorTask := NewMockWork("error", "error task", func(ctx context.Context) error {
		errorTaskExecuted = true
		return errors.New("test error")
	})

	longTaskExecuted := false
	longTask := NewMockWork("long", "long task", func(ctx context.Context) error {
		select {
		case <-time.After(500 * time.Millisecond):
			longTaskExecuted = true
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	// Submit tasks
	pool.Submit(successfulTask)
	pool.Submit(errorTask)
	pool.Submit(longTask)

	// Wait for tasks to be processed
	time.Sleep(700 * time.Millisecond)

	// Check if all tasks were executed
	if !successfulTaskExecuted {
		t.Error("Successful task was not executed")
	}
	if !errorTaskExecuted {
		t.Error("Error task was not executed")
	}
	if !longTaskExecuted {
		t.Error("Long task was not executed")
	}

	// Shutdown the pool
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	pool.Shutdown(ctx)
}

func TestPoolGracefulShutdown(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logger.New(logger.INFO)
	pool, _ := NewPool(parentCtx, PoolConfig{
		Workers:   2,
		QueueSize: 5,
		Logger:    log,
	})

	pool.Start()

	// Create a task that takes long enough to execute
	finished := false
	longTask := NewMockWork("long-shutdown", "long shutdown task", func(ctx context.Context) error {
		// Use a shorter time to ensure it completes before shutdown
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-timer.C:
			finished = true
			return nil
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		}
	})

	// Submit the task
	pool.Submit(longTask)

	// Give the task enough time to complete before shutting down
	time.Sleep(200 * time.Millisecond)

	// Initiate shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := pool.Shutdown(ctx)
	if err != nil {
		t.Errorf("Error during graceful shutdown: %v", err)
	}

	// Task should have completed
	if !finished {
		t.Error("Task did not finish during graceful shutdown")
	}
}

func TestPoolShutdownTimeout(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logger.New(logger.INFO)
	pool, _ := NewPool(parentCtx, PoolConfig{
		Workers:   1,
		QueueSize: 5,
		Logger:    log,
	})

	pool.Start()

	// Create a task that blocks indefinitely without checking context
	// This forces the shutdown to timeout
	blockForever := make(chan struct{})
	veryLongTask := NewMockWork("timeout", "timeout task", func(ctx context.Context) error {
		// This will block forever until the test ends
		<-blockForever
		return nil
	})

	// Submit the task and make sure it starts executing
	pool.Submit(veryLongTask)
	time.Sleep(50 * time.Millisecond) // Give time for the task to start

	// Set a very short timeout for shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	// This should timeout since the task will block forever
	err := pool.Shutdown(shutdownCtx)

	// We expect an error
	if err == nil {
		t.Error("Expected timeout error during shutdown, got nil")
	}

	// Clean up
	close(blockForever) // Unblock the task

	// Properly clean up with a longer timeout
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cleanupCancel()
	pool.Shutdown(cleanupCtx)
}

func TestTrySubmit(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logger.New(logger.INFO)
	pool, _ := NewPool(parentCtx, PoolConfig{
		Workers:   1,
		QueueSize: 2, // Very small queue for testing
		Logger:    log,
	})

	pool.Start()

	// Create tasks that block the worker
	blockTask := NewMockWork("block", "blocking task", func(ctx context.Context) error {
		// This will block the worker
		time.Sleep(200 * time.Millisecond)
		return nil
	})

	// Submit blocking task
	pool.Submit(blockTask)

	// Fill the queue
	for i := 0; i < 2; i++ {
		task := NewMockWork("fill", "fill queue task", func(ctx context.Context) error {
			return nil
		})
		err := pool.Submit(task)
		if err != nil {
			t.Errorf("Error filling queue: %v", err)
		}
	}

	// Try to submit when queue is full
	overflowTask := NewMockWork("overflow", "overflow task", func(ctx context.Context) error {
		return nil
	})

	// Regular Submit would block, but TrySubmit should return an error
	err := pool.TrySubmit(overflowTask)
	if err == nil {
		t.Error("Expected error when trying to submit to full queue")
	}

	// Let everything finish
	time.Sleep(300 * time.Millisecond)

	// Shutdown the pool
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	pool.Shutdown(ctx)
}
