package generator

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/user/chasqui/pkg/logger"
	"github.com/user/chasqui/pkg/modules/hephaestus"
)

// SleepTask is a simple task that sleeps for a specified duration.
type SleepTask struct {
	id       string
	name     string
	duration time.Duration
	log      *logger.Logger
}

// NewSleepTask creates a new SleepTask with the given duration.
func NewSleepTask(duration time.Duration, log *logger.Logger) hephaestus.Work {
	taskID := uuid.New().String()
	return &SleepTask{
		id:       taskID,
		name:     "SleepTask",
		duration: duration,
		log:      log,
	}
}

// Execute implements the hephaestus.Work interface.
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

// ID implements the hephaestus.Work interface.
func (t *SleepTask) ID() string {
	return t.id
}

// Name implements the hephaestus.Work interface.
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

// NewFailingTask creates a new FailingTask with the given failure probability.
func NewFailingTask(failProb float64, executionDur time.Duration, log *logger.Logger) hephaestus.Work {
	taskID := uuid.New().String()
	return &FailingTask{
		id:           taskID,
		name:         "FailingTask",
		failProb:     failProb,
		executionDur: executionDur,
		log:          log,
	}
}

// Execute implements the hephaestus.Work interface.
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

// ID implements the hephaestus.Work interface.
func (t *FailingTask) ID() string {
	return t.id
}

// Name implements the hephaestus.Work interface.
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

// NewProcessingTask creates a new ProcessingTask with the given parameters.
func NewProcessingTask(name string, items int, processTime time.Duration, log *logger.Logger) hephaestus.Work {
	taskID := uuid.New().String()
	return &ProcessingTask{
		id:          taskID,
		name:        name,
		items:       items,
		processTime: processTime,
		log:         log,
	}
}

// Execute implements the hephaestus.Work interface.
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

// ID implements the hephaestus.Work interface.
func (t *ProcessingTask) ID() string {
	return t.id
}

// Name implements the hephaestus.Work interface.
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
