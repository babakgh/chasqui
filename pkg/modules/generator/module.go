package generator

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/user/chasqui/pkg/logger"
	"github.com/user/chasqui/pkg/module"
	"github.com/user/chasqui/pkg/modules/hephaestus"
)

// Module represents the task generator module.
type Module struct {
	ctx         context.Context
	cancel      context.CancelFunc
	log         *logger.Logger
	config      Config
	wg          sync.WaitGroup
	hephaestus  hephaestus.Module
	taskChannel chan<- hephaestus.Work
}

// Config holds the configuration for the generator module.
type Config struct {
	// Count is the number of tasks to generate
	Count int
	// Interval is the time to wait between generating tasks
	Interval time.Duration
	// MinDuration is the minimum duration of a task
	MinDuration time.Duration
	// MaxDuration is the maximum duration of a task
	MaxDuration time.Duration
	// FailProb is the probability of a task failing
	FailProb float64
}

// NewModule creates a new generator module.
func NewModule(ctx context.Context, log *logger.Logger, config Config) (module.Module, error) {
	if config.Count < 0 {
		return nil, fmt.Errorf("count must be non-negative")
	}
	if config.Interval < time.Millisecond {
		return nil, fmt.Errorf("interval must be at least 1ms")
	}
	if config.MinDuration < time.Millisecond {
		return nil, fmt.Errorf("min duration must be at least 1ms")
	}
	if config.MaxDuration < config.MinDuration {
		return nil, fmt.Errorf("max duration must be greater than or equal to min duration")
	}
	if config.FailProb < 0 || config.FailProb > 1 {
		return nil, fmt.Errorf("fail probability must be between 0 and 1")
	}

	ctx, cancel := context.WithCancel(ctx)

	return &Module{
		ctx:    ctx,
		cancel: cancel,
		log:    log,
		config: config,
	}, nil
}

// Name returns the name of the module.
func (m *Module) Name() string {
	return "generator"
}

// Init initializes the module with the provided configuration.
func (m *Module) Init(cfg any) error {
	m.log.Debug("Initializing generator module")
	return nil
}

// Start starts the task generation process.
func (m *Module) Start() error {
	m.log.Info("Starting generator module")

	// Get the Hephaestus module
	reg := module.GetRegistry()
	mod, err := reg.GetModule("hephaestus")
	if err != nil {
		return fmt.Errorf("failed to get hephaestus module: %w", err)
	}

	// Type assertion to get the Hephaestus module
	heph, ok := mod.(*hephaestus.Module)
	if !ok {
		return fmt.Errorf("module 'hephaestus' is not a hephaestus.Module")
	}

	// Store the Hephaestus module and its task channel
	m.hephaestus = *heph
	m.taskChannel = heph.TaskChannel()

	// Start generating tasks
	m.wg.Add(1)
	go m.generateTasks()

	// Block until context is done
	<-m.ctx.Done()
	return nil
}

// Stop stops the task generation process.
func (m *Module) Stop(ctx context.Context) error {
	m.log.Info("Stopping generator module")
	m.cancel()

	// Wait for the generator goroutine to finish
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		m.log.Info("Generator module stopped successfully")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("generator shutdown timed out: %w", ctx.Err())
	}
}

// Dependencies returns the modules this module depends on.
func (m *Module) Dependencies() []string {
	return []string{"hephaestus"}
}

// generateTasks generates and submits tasks to the Hephaestus module.
func (m *Module) generateTasks() {
	defer m.wg.Done()

	m.log.Info("Starting task generation: will generate %d tasks", m.config.Count)
	m.log.Info("Task parameters - Interval: %v, MinDuration: %v, MaxDuration: %v, FailProb: %.2f",
		m.config.Interval, m.config.MinDuration, m.config.MaxDuration, m.config.FailProb)

	for i := 0; i < m.config.Count; i++ {
		select {
		case <-m.ctx.Done():
			m.log.Info("Task generation cancelled after generating %d/%d tasks", i, m.config.Count)
			return
		case <-time.After(m.config.Interval):
			// Generate and submit a task
			task := m.createRandomTask()
			if err := m.hephaestus.Submit(task); err != nil {
				m.log.Error("Failed to submit task %d/%d: %v", i+1, m.config.Count, err)
			} else {
				m.log.Debug("Generated and submitted task %d/%d: %s (ID: %s)",
					i+1, m.config.Count, task.Name(), task.ID())
			}
		}
	}

	m.log.Info("Finished generating all %d tasks", m.config.Count)
}

// createRandomTask creates a random task.
func (m *Module) createRandomTask() hephaestus.Work {
	// Choose a random task type
	taskType := rand.Intn(3)

	switch taskType {
	case 0:
		// Sleep task
		duration := randomDuration(m.config.MinDuration, m.config.MaxDuration)
		return NewSleepTask(duration, m.log)
	case 1:
		// Failing task
		duration := randomDuration(m.config.MinDuration, m.config.MaxDuration)
		return NewFailingTask(m.config.FailProb, duration, m.log)
	default:
		// Processing task
		items := rand.Intn(10) + 1
		duration := randomDuration(m.config.MinDuration, m.config.MaxDuration)
		taskName := fmt.Sprintf("Process-%s", uuid.New().String()[:8])
		return NewProcessingTask(taskName, items, duration, m.log)
	}
}

// randomDuration returns a random duration between min and max.
func randomDuration(min, max time.Duration) time.Duration {
	return min + time.Duration(rand.Int63n(int64(max-min)))
}
