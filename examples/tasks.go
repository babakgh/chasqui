package examples

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/user/chasqui/pkg/logger"
	"github.com/user/chasqui/pkg/worker"
)

// SleepTask is a simple task that sleeps for a specified duration.
type SleepTask struct {
	id       string
	name     string
	duration time.Duration
	log      *logger.Logger
}

// NewSleepTask creates a new SleepTask with the given duration.
func NewSleepTask(duration time.Duration, log *logger.Logger) *SleepTask {
	taskID := uuid.New().String()
	if log == nil {
		log = logger.New(logger.INFO)
	}

	log.Debug("Created SleepTask (ID: %s) with duration %v", taskID, duration)

	return &SleepTask{
		id:       taskID,
		name:     "SleepTask",
		duration: duration,
		log:      log,
	}
}

// Execute implements the worker.Work interface.
func (t *SleepTask) Execute(ctx context.Context) error {
	t.log.Debug("Starting SleepTask execution (ID: %s, duration: %v)", t.id, t.duration)
	startTime := time.Now()

	select {
	case <-ctx.Done():
		elapsed := time.Since(startTime)
		t.log.Debug("SleepTask (ID: %s) cancelled after %v: %v", t.id, elapsed, ctx.Err())
		return ctx.Err()
	case <-time.After(t.duration):
		elapsed := time.Since(startTime)
		t.log.Debug("SleepTask (ID: %s) completed successfully after %v", t.id, elapsed)
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
	log          *logger.Logger
}

// NewFailingTask creates a new FailingTask with the given failure probability and execution duration.
func NewFailingTask(failProb float64, executionDur time.Duration, log *logger.Logger) *FailingTask {
	taskID := uuid.New().String()
	if log == nil {
		log = logger.New(logger.INFO)
	}

	log.Debug("Created FailingTask (ID: %s) with failProb %.2f and duration %v",
		taskID, failProb, executionDur)

	return &FailingTask{
		id:           taskID,
		name:         "FailingTask",
		failProb:     failProb,
		executionDur: executionDur,
		log:          log,
	}
}

// Execute implements the worker.Work interface.
func (t *FailingTask) Execute(ctx context.Context) error {
	t.log.Debug("Starting FailingTask execution (ID: %s, duration: %v, failProb: %.2f)",
		t.id, t.executionDur, t.failProb)
	startTime := time.Now()

	select {
	case <-ctx.Done():
		elapsed := time.Since(startTime)
		t.log.Debug("FailingTask (ID: %s) cancelled after %v: %v", t.id, elapsed, ctx.Err())
		return ctx.Err()
	case <-time.After(t.executionDur):
		elapsed := time.Since(startTime)

		failRoll := rand.Float64()
		willFail := failRoll < t.failProb

		if willFail {
			t.log.Debug("FailingTask (ID: %s) failing with probability %.2f (roll: %.2f) after %v",
				t.id, t.failProb, failRoll, elapsed)
			return fmt.Errorf("task failed with probability %f", t.failProb)
		}

		t.log.Debug("FailingTask (ID: %s) completed successfully (roll: %.2f > failProb: %.2f) after %v",
			t.id, failRoll, t.failProb, elapsed)
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
	log            *logger.Logger
}

// NewProcessingTask creates a new ProcessingTask that processes the given number of items.
func NewProcessingTask(name string, items int, processTime time.Duration, log *logger.Logger) *ProcessingTask {
	taskID := uuid.New().String()
	if log == nil {
		log = logger.New(logger.INFO)
	}

	log.Debug("Created ProcessingTask %s (ID: %s) with %d items and total duration %v",
		name, taskID, items, processTime)

	return &ProcessingTask{
		id:          taskID,
		name:        name,
		items:       items,
		processTime: processTime,
		log:         log,
	}
}

// Execute implements the worker.Work interface.
func (t *ProcessingTask) Execute(ctx context.Context) error {
	t.log.Debug("Starting ProcessingTask %s execution (ID: %s, items: %d, duration: %v)",
		t.name, t.id, t.items, t.processTime)
	startTime := time.Now()

	// Reset processed items to handle potential reuse
	t.processedItems = 0

	// Calculate duration per item
	itemDuration := t.processTime / time.Duration(t.items)
	t.log.Debug("ProcessingTask %s (ID: %s) processing each item for %v",
		t.name, t.id, itemDuration)

	for i := 0; i < t.items; i++ {
		itemStart := time.Now()
		select {
		case <-ctx.Done():
			elapsed := time.Since(startTime)
			t.log.Debug("ProcessingTask %s (ID: %s) cancelled after processing %d/%d items (%v): %v",
				t.name, t.id, t.processedItems, t.items, elapsed, ctx.Err())
			return ctx.Err()
		case <-time.After(itemDuration):
			t.processedItems++
			itemElapsed := time.Since(itemStart)
			progress := t.GetProgress()
			t.log.Debug("ProcessingTask %s (ID: %s) processed item %d/%d (%.1f%%) in %v",
				t.name, t.id, t.processedItems, t.items, progress, itemElapsed)
		}
	}

	elapsed := time.Since(startTime)
	t.log.Debug("ProcessingTask %s (ID: %s) completed all %d items successfully in %v",
		t.name, t.id, t.items, elapsed)
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
	log          *logger.Logger
}

// NewTaskGenerator creates a new TaskGenerator with the given parameters.
func NewTaskGenerator(log *logger.Logger) *TaskGenerator {
	if log == nil {
		log = logger.New(logger.INFO)
	}

	// Reduce sleep times as specified in the prompt
	return &TaskGenerator{
		MinSleep:     10 * time.Millisecond,  // Reduced from 100ms
		MaxSleep:     200 * time.Millisecond, // Reduced from 2s
		FailProb:     0.2,
		ProcessItems: 10,
		taskChan:     make(chan worker.Work, 100),
		log:          log,
	}
}

// Generate starts generating tasks and sends them to the returned channel.
// It will generate the specified number of tasks and then close the channel.
func (g *TaskGenerator) Generate(ctx context.Context, count int) <-chan worker.Work {
	g.log.Info("Starting task generation: will generate %d tasks", count)
	g.log.Info("Task parameters - MinSleep: %v, MaxSleep: %v, FailProb: %.2f, ProcessItems: %d",
		g.MinSleep, g.MaxSleep, g.FailProb, g.ProcessItems)

	go func() {
		defer close(g.taskChan)
		g.log.Debug("Task generator goroutine started")

		for i := 0; i < count; i++ {
			select {
			case <-ctx.Done():
				g.log.Debug("Task generation cancelled after generating %d/%d tasks: %v", i, count, ctx.Err())
				return
			default:
				// Generate a random task type
				taskType := rand.Intn(3)
				var task worker.Work

				switch taskType {
				case 0:
					// Sleep task
					duration := g.MinSleep + time.Duration(rand.Int63n(int64(g.MaxSleep-g.MinSleep)))
					g.log.Debug("Generating SleepTask %d/%d with duration %v", i+1, count, duration)
					task = NewSleepTask(duration, g.log)
				case 1:
					// Failing task
					duration := g.MinSleep + time.Duration(rand.Int63n(int64(g.MaxSleep-g.MinSleep)))
					g.log.Debug("Generating FailingTask %d/%d with duration %v and failProb %.2f",
						i+1, count, duration, g.FailProb)
					task = NewFailingTask(g.FailProb, duration, g.log)
				case 2:
					// Processing task
					items := rand.Intn(g.ProcessItems) + 1
					duration := g.MinSleep + time.Duration(rand.Int63n(int64(g.MaxSleep-g.MinSleep)))
					taskName := fmt.Sprintf("Process-%d", i)
					g.log.Debug("Generating ProcessingTask %d/%d (%s) with %d items and duration %v",
						i+1, count, taskName, items, duration)
					task = NewProcessingTask(taskName, items, duration, g.log)
				}

				// Send the task to the channel
				select {
				case g.taskChan <- task:
					g.log.Debug("Task %d/%d sent to channel: %s (ID: %s)",
						i+1, count, task.Name(), task.ID())
				case <-ctx.Done():
					g.log.Debug("Task generation cancelled while sending task %d/%d: %v",
						i+1, count, ctx.Err())
					return
				}

				// Small delay between generating tasks
				delay := 50 * time.Millisecond
				g.log.Debug("Waiting %v before generating next task", delay)
				time.Sleep(delay)
			}
		}

		g.log.Info("Completed generating all %d tasks", count)
	}()

	return g.taskChan
}
