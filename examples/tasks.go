package examples

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/user/chasqui/pkg/worker"
)

// SleepTask is a simple task that sleeps for a specified duration.
type SleepTask struct {
	id       string
	name     string
	duration time.Duration
}

// NewSleepTask creates a new SleepTask with the given duration.
func NewSleepTask(duration time.Duration) *SleepTask {
	return &SleepTask{
		id:       uuid.New().String(),
		name:     "SleepTask",
		duration: duration,
	}
}

// Execute implements the worker.Work interface.
func (t *SleepTask) Execute(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(t.duration):
		return nil
	}
}

// ID implements the worker.Work interface.
func (t *SleepTask) ID() string {
	return t.id
}

// Name implements the worker.Work interface.
func (t *SleepTask) Name() string {
	return t.name
}

// FailingTask is a task that has a configurable probability of failing.
type FailingTask struct {
	id           string
	name         string
	failProb     float64
	executionDur time.Duration
}

// NewFailingTask creates a new FailingTask with the given failure probability and execution duration.
func NewFailingTask(failProb float64, executionDur time.Duration) *FailingTask {
	return &FailingTask{
		id:           uuid.New().String(),
		name:         "FailingTask",
		failProb:     failProb,
		executionDur: executionDur,
	}
}

// Execute implements the worker.Work interface.
func (t *FailingTask) Execute(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(t.executionDur):
		if rand.Float64() < t.failProb {
			return fmt.Errorf("task failed with probability %f", t.failProb)
		}
		return nil
	}
}

// ID implements the worker.Work interface.
func (t *FailingTask) ID() string {
	return t.id
}

// Name implements the worker.Work interface.
func (t *FailingTask) Name() string {
	return t.name
}

// ProcessingTask is a task that simulates data processing with progress reporting.
type ProcessingTask struct {
	id             string
	name           string
	items          int
	processTime    time.Duration
	processedItems int
}

// NewProcessingTask creates a new ProcessingTask that processes the given number of items.
func NewProcessingTask(name string, items int, processTime time.Duration) *ProcessingTask {
	return &ProcessingTask{
		id:          uuid.New().String(),
		name:        name,
		items:       items,
		processTime: processTime,
	}
}

// Execute implements the worker.Work interface.
func (t *ProcessingTask) Execute(ctx context.Context) error {
	itemDuration := t.processTime / time.Duration(t.items)

	for i := 0; i < t.items; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(itemDuration):
			t.processedItems++
		}
	}

	return nil
}

// ID implements the worker.Work interface.
func (t *ProcessingTask) ID() string {
	return t.id
}

// Name implements the worker.Work interface.
func (t *ProcessingTask) Name() string {
	return t.name
}

// GetProgress returns the progress of the task as a percentage.
func (t *ProcessingTask) GetProgress() float64 {
	if t.items == 0 {
		return 100.0
	}
	return float64(t.processedItems) / float64(t.items) * 100.0
}

// TaskGenerator helps generate a stream of tasks for testing.
type TaskGenerator struct {
	MinSleep     time.Duration
	MaxSleep     time.Duration
	FailProb     float64
	ProcessItems int
	taskChan     chan worker.Work
}

// NewTaskGenerator creates a new TaskGenerator with the given parameters.
func NewTaskGenerator() *TaskGenerator {
	return &TaskGenerator{
		MinSleep:     100 * time.Millisecond,
		MaxSleep:     2 * time.Second,
		FailProb:     0.2,
		ProcessItems: 10,
		taskChan:     make(chan worker.Work, 100),
	}
}

// Generate starts generating tasks and sends them to the returned channel.
// It will generate the specified number of tasks and then close the channel.
func (g *TaskGenerator) Generate(ctx context.Context, count int) <-chan worker.Work {
	go func() {
		defer close(g.taskChan)

		for i := 0; i < count; i++ {
			select {
			case <-ctx.Done():
				return
			default:
				// Generate a random task type
				taskType := rand.Intn(3)
				var task worker.Work

				switch taskType {
				case 0:
					// Sleep task
					duration := g.MinSleep + time.Duration(rand.Int63n(int64(g.MaxSleep-g.MinSleep)))
					task = NewSleepTask(duration)
				case 1:
					// Failing task
					duration := g.MinSleep + time.Duration(rand.Int63n(int64(g.MaxSleep-g.MinSleep)))
					task = NewFailingTask(g.FailProb, duration)
				case 2:
					// Processing task
					items := rand.Intn(g.ProcessItems) + 1
					duration := g.MinSleep + time.Duration(rand.Int63n(int64(g.MaxSleep-g.MinSleep)))
					task = NewProcessingTask(fmt.Sprintf("Process-%d", i), items, duration)
				}

				g.taskChan <- task

				// Small delay between generating tasks
				time.Sleep(50 * time.Millisecond)
			}
		}
	}()

	return g.taskChan
}
