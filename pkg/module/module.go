package module

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/user/chasqui/pkg/logger"
)

// Duration wraps time.Duration for easier handling in configuration
type Duration struct {
	Duration time.Duration
}

// Module is the interface that all modules must implement
type Module interface {
	// Name returns the module's name
	Name() string

	// Init initializes the module with the provided config
	Init(cfg any) error

	// Start starts the module (and blocks until error or Stop is called)
	Start() error

	// Stop gracefully stops the module
	Stop(ctx context.Context) error

	// Dependencies returns other module names this module depends on
	Dependencies() []string
}

// Registry manages module registration and lifecycle
type Registry struct {
	modules     map[string]Module
	initialized map[string]bool
	started     map[string]bool
	log         *logger.Logger
	mu          sync.RWMutex
}

// Global registry instance
var (
	globalRegistry *Registry
	registryOnce   sync.Once
)

// GetRegistry returns the global registry instance
func GetRegistry() *Registry {
	registryOnce.Do(func() {
		globalRegistry = NewRegistry(logger.New(logger.INFO))
	})
	return globalRegistry
}

// SetRegistry sets the global registry instance (useful for testing)
func SetRegistry(r *Registry) {
	globalRegistry = r
}

// NewRegistry creates a new module registry
func NewRegistry(log *logger.Logger) *Registry {
	if log == nil {
		log = logger.New(logger.INFO)
	}

	return &Registry{
		modules:     make(map[string]Module),
		initialized: make(map[string]bool),
		started:     make(map[string]bool),
		log:         log,
	}
}

// Register registers a module with the registry
func (r *Registry) Register(m Module) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := m.Name()
	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("module %s is already registered", name)
	}

	r.log.Info("Registering module: %s", name)
	r.modules[name] = m
	return nil
}

// GetModule returns a module by name
func (r *Registry) GetModule(name string) (Module, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, exists := r.modules[name]
	if !exists {
		return nil, fmt.Errorf("module %s not found", name)
	}

	return m, nil
}

// InitModule initializes a module and its dependencies
func (r *Registry) InitModule(name string, cfg any) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.initModule(name, cfg)
}

// Internal method to initialize a module and its dependencies
func (r *Registry) initModule(name string, cfg any) error {
	// Check if module exists
	module, exists := r.modules[name]
	if !exists {
		return fmt.Errorf("module %s not found", name)
	}

	// Check if already initialized
	if r.initialized[name] {
		return nil
	}

	// Initialize dependencies first
	for _, depName := range module.Dependencies() {
		if err := r.initModule(depName, cfg); err != nil {
			return fmt.Errorf("failed to initialize dependency %s for module %s: %w", depName, name, err)
		}
	}

	// Initialize the module
	r.log.Info("Initializing module: %s", name)
	if err := module.Init(cfg); err != nil {
		return fmt.Errorf("failed to initialize module %s: %w", name, err)
	}

	r.initialized[name] = true
	return nil
}

// StartModule starts a module and its dependencies
func (r *Registry) StartModule(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.startModule(name)
}

// Internal method to start a module and its dependencies
func (r *Registry) startModule(name string) error {
	// Check if module exists
	module, exists := r.modules[name]
	if !exists {
		return fmt.Errorf("module %s not found", name)
	}

	// Check if already started
	if r.started[name] {
		return nil
	}

	// Check if initialized
	if !r.initialized[name] {
		return fmt.Errorf("module %s is not initialized", name)
	}

	// Start dependencies first
	for _, depName := range module.Dependencies() {
		if err := r.startModule(depName); err != nil {
			return fmt.Errorf("failed to start dependency %s for module %s: %w", depName, name, err)
		}
	}

	// Start the module in a separate goroutine
	r.log.Info("Starting module: %s", name)

	// Mark as started before actual start to prevent circular dependencies
	r.started[name] = true

	go func() {
		if err := module.Start(); err != nil {
			r.log.Error("Module %s encountered an error: %v", name, err)
			r.started[name] = false
		}
	}()

	return nil
}

// StopModule stops a module
func (r *Registry) StopModule(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if module exists
	module, exists := r.modules[name]
	if !exists {
		return fmt.Errorf("module %s not found", name)
	}

	// Check if started
	if !r.started[name] {
		return nil
	}

	// Stop the module
	r.log.Info("Stopping module: %s", name)
	if err := module.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop module %s: %w", name, err)
	}

	r.started[name] = false
	return nil
}

// StopAllModules stops all modules in reverse dependency order
func (r *Registry) StopAllModules(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Build a list of modules to stop (all started modules)
	var modulesToStop []string
	for name, started := range r.started {
		if started {
			modulesToStop = append(modulesToStop, name)
		}
	}

	// Stop modules in reverse dependency order (basic implementation)
	// A more sophisticated implementation would build a true dependency graph
	for i := len(modulesToStop) - 1; i >= 0; i-- {
		name := modulesToStop[i]
		module := r.modules[name]

		r.log.Info("Stopping module: %s", name)
		if err := module.Stop(ctx); err != nil {
			r.log.Error("Failed to stop module %s: %v", name, err)
			// Continue stopping other modules
		}

		r.started[name] = false
	}

	return nil
}

// GetModuleNames returns a list of all registered module names
func (r *Registry) GetModuleNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.modules))
	for name := range r.modules {
		names = append(names, name)
	}

	return names
}

// IsModuleStarted checks if a module is started
func (r *Registry) IsModuleStarted(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.started[name]
}

// IsModuleInitialized checks if a module is initialized
func (r *Registry) IsModuleInitialized(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.initialized[name]
}
