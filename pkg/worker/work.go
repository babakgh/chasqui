package worker

import "context"

// Work is an interface that must be implemented by any task
// that needs to be executed by the worker pool.
type Work interface {
	// Execute performs the actual task work.
	// It receives a context for cancellation and returns an error if the task fails.
	Execute(ctx context.Context) error

	// ID returns a unique identifier for the task.
	ID() string

	// Name returns a human-readable name for the task.
	Name() string
}
