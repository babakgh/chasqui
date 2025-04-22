package hephaestus

import (
	"context"
	"testing"
	"time"

	"github.com/user/chasqui/pkg/logger"
)

func TestDispatcherCreation(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logger.New(logger.INFO)

	// Create a pool for the dispatcher to use
	pool, _ := NewPool(parentCtx, PoolConfig{
		Workers:   2,
		QueueSize: 10,
		Logger:    log,
	})
	pool.Start()

	// Test with invalid configuration
	testCases := []struct {
		name   string
		config DispatcherConfig
	}{
		{
			name: "nil pool",
			config: DispatcherConfig{
				Pool:      nil,
				Logger:    log,
				BatchSize: 5,
			},
		},
		{
			name: "nil logger",
			config: DispatcherConfig{
				Pool:      pool,
				Logger:    nil,
				BatchSize: 5,
			},
		},
		{
			name: "zero batch size",
			config: DispatcherConfig{
				Pool:      pool,
				Logger:    log,
				BatchSize: 0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewDispatcher(parentCtx, tc.config)
			if err == nil {
				t.Errorf("Expected error for invalid config %s, got nil", tc.name)
			}
		})
	}

	// Test with valid configuration
	dispatcher, err := NewDispatcher(parentCtx, DispatcherConfig{
		Pool:      pool,
		Logger:    log,
		BatchSize: 5,
	})

	if err != nil {
		t.Fatalf("Error creating dispatcher with valid config: %v", err)
	}

	if dispatcher == nil {
		t.Fatal("Dispatcher should not be nil with valid config")
	}

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	pool.Shutdown(ctx)
}

func TestDispatcherStartAndShutdown(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logger.New(logger.INFO)

	// Create a pool
	pool, _ := NewPool(parentCtx, PoolConfig{
		Workers:   2,
		QueueSize: 10,
		Logger:    log,
	})
	pool.Start()

	// Create a dispatcher
	dispatcher, _ := NewDispatcher(parentCtx, DispatcherConfig{
		Pool:      pool,
		Logger:    log,
		BatchSize: 5,
	})

	// Try operations before starting
	mockTasks := []Work{
		NewMockWork("1", "test1", func(ctx context.Context) error { return nil }),
	}

	_, err := dispatcher.DispatchBatch(mockTasks)
	if err == nil {
		t.Error("Should not be able to dispatch batch before starting")
	}

	err = dispatcher.ProcessChannel(make(chan Work))
	if err == nil {
		t.Error("Should not be able to process channel before starting")
	}

	// Start the dispatcher
	err = dispatcher.Start()
	if err != nil {
		t.Fatalf("Error starting dispatcher: %v", err)
	}

	// Try starting again
	err = dispatcher.Start()
	if err == nil {
		t.Error("Should not be able to start dispatcher twice")
	}

	// Shutdown the dispatcher
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = dispatcher.Shutdown(ctx)
	if err != nil {
		t.Errorf("Error shutting down dispatcher: %v", err)
	}

	// Try operations after shutdown
	_, err = dispatcher.DispatchBatch(mockTasks)
	if err == nil {
		t.Error("Should not be able to dispatch batch after shutdown")
	}

	// Clean up pool
	pool.Shutdown(ctx)
}

func TestDispatchBatch(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logger.New(logger.INFO)

	// Create a pool
	pool, _ := NewPool(parentCtx, PoolConfig{
		Workers:   2,
		QueueSize: 10,
		Logger:    log,
	})
	pool.Start()

	// Create a dispatcher
	dispatcher, _ := NewDispatcher(parentCtx, DispatcherConfig{
		Pool:      pool,
		Logger:    log,
		BatchSize: 5,
	})
	dispatcher.Start()

	// Create a batch of tasks
	taskCount := 5
	executed := make([]bool, taskCount)
	tasks := make([]Work, taskCount)

	for i := 0; i < taskCount; i++ {
		index := i // Capture the index for closure
		tasks[i] = NewMockWork(
			"batch-task",
			"batch task",
			func(ctx context.Context) error {
				executed[index] = true
				return nil
			},
		)
	}

	// Dispatch the batch
	submitted, err := dispatcher.DispatchBatch(tasks)
	if err != nil {
		t.Errorf("Error dispatching batch: %v", err)
	}
	if submitted != taskCount {
		t.Errorf("Expected %d tasks to be submitted, got %d", taskCount, submitted)
	}

	// Wait for tasks to be processed
	time.Sleep(200 * time.Millisecond)

	// Check if all tasks were executed
	for i, wasExecuted := range executed {
		if !wasExecuted {
			t.Errorf("Task %d was not executed", i)
		}
	}

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	dispatcher.Shutdown(ctx)
	pool.Shutdown(ctx)
}

func TestProcessChannel(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logger.New(logger.INFO)

	// Create a pool
	pool, _ := NewPool(parentCtx, PoolConfig{
		Workers:   2,
		QueueSize: 10,
		Logger:    log,
	})
	pool.Start()

	// Create a dispatcher with small batch size to test batching
	batchSize := 3
	dispatcher, _ := NewDispatcher(parentCtx, DispatcherConfig{
		Pool:      pool,
		Logger:    log,
		BatchSize: batchSize,
	})
	dispatcher.Start()

	// Create a channel and tasks
	taskCount := 10
	taskChan := make(chan Work, taskCount)
	executed := make([]bool, taskCount)

	for i := 0; i < taskCount; i++ {
		index := i // Capture the index for closure
		taskChan <- NewMockWork(
			"channel-task",
			"channel task",
			func(ctx context.Context) error {
				executed[index] = true
				return nil
			},
		)
	}
	close(taskChan)

	// Process the channel
	err := dispatcher.ProcessChannel(taskChan)
	if err != nil {
		t.Errorf("Error processing channel: %v", err)
	}

	// Wait for all tasks to be processed
	time.Sleep(500 * time.Millisecond)

	// Check if all tasks were executed
	for i, wasExecuted := range executed {
		if !wasExecuted {
			t.Errorf("Task %d was not executed", i)
		}
	}

	// Clean up
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	dispatcher.Shutdown(ctx)
	pool.Shutdown(ctx)
}

func TestDispatcherGracefulShutdown(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	log := logger.New(logger.INFO)

	// Create a pool
	pool, _ := NewPool(parentCtx, PoolConfig{
		Workers:   2,
		QueueSize: 10,
		Logger:    log,
	})
	pool.Start()

	// Create a dispatcher
	dispatcher, _ := NewDispatcher(parentCtx, DispatcherConfig{
		Pool:      pool,
		Logger:    log,
		BatchSize: 2,
	})
	dispatcher.Start()

	// Create a channel for tasks and a cancellation channel
	taskChan := make(chan Work)
	stopChan := make(chan struct{})

	go func() {
		// Send tasks until signaled to stop
		i := 0
		for {
			select {
			case <-stopChan:
				// Received stop signal, exit goroutine
				return
			default:
				select {
				case taskChan <- NewMockWork(
					"continuous-task",
					"continuous task",
					func(ctx context.Context) error {
						// Small delay to simulate work
						time.Sleep(50 * time.Millisecond)
						return nil
					},
				):
					i++
				case <-stopChan:
					// Also check in the default case
					return
				case <-time.After(500 * time.Millisecond):
					// Channel might be blocked, but don't hang forever
					return
				}
			}
		}
	}()

	// Process the channel
	dispatcher.ProcessChannel(taskChan)

	// Let some tasks be processed
	time.Sleep(200 * time.Millisecond)

	// Signal the task producer to stop before closing the channel
	close(stopChan)

	// Wait a bit to ensure the goroutine has exited
	time.Sleep(50 * time.Millisecond)

	// Now it's safe to close the channel
	close(taskChan)

	// Shutdown gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	err := dispatcher.Shutdown(ctx)
	if err != nil {
		t.Errorf("Error during graceful shutdown: %v", err)
	}

	// Clean up pool
	pool.Shutdown(ctx)
}
