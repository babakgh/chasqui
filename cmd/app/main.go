package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/user/chasqui/pkg/config"
	"github.com/user/chasqui/pkg/logger"
	"github.com/user/chasqui/pkg/module"
	"github.com/user/chasqui/pkg/modules/generator"
	"github.com/user/chasqui/pkg/modules/hephaestus"
)

func main() {
	// Create logger
	log := logger.New(logger.INFO)
	log.Info("Starting Chasqui application")

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatal("Failed to load configuration: %v", err)
	}

	// Print configuration
	config.PrintConfig(cfg)

	// Create application context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize module registry
	registry := module.NewRegistry(log)
	module.SetRegistry(registry)

	// Register modules based on configuration
	if err := registerModules(ctx, log, cfg, registry); err != nil {
		log.Fatal("Failed to register modules: %v", err)
	}

	// Initialize modules
	for _, moduleName := range cfg.EnabledModules {
		if err := registry.InitModule(moduleName, cfg); err != nil {
			log.Fatal("Failed to initialize module %s: %v", moduleName, err)
		}
	}

	// Start modules
	for _, moduleName := range cfg.EnabledModules {
		if err := registry.StartModule(moduleName); err != nil {
			log.Fatal("Failed to start module %s: %v", moduleName, err)
		}
	}

	log.Info("All modules started successfully")

	// Set up signal handling for graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for termination signal
	sig := <-signalChan
	log.Info("Received signal %v, shutting down...", sig)

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Stop all modules
	if err := registry.StopAllModules(shutdownCtx); err != nil {
		log.Error("Error stopping modules: %v", err)
		os.Exit(1)
	}

	log.Info("Chasqui application stopped successfully")
}

// registerModules registers all available modules with the registry
func registerModules(ctx context.Context, log *logger.Logger, cfg *config.Config, registry *module.Registry) error {
	// Define module factories
	factories := map[string]func(context.Context, *logger.Logger, *config.Config) (module.Module, error){
		"hephaestus": hephaestus.Factory,
		"generator":  generator.Factory,
	}

	// Register enabled modules
	for _, moduleName := range cfg.EnabledModules {
		factory, exists := factories[moduleName]
		if !exists {
			return fmt.Errorf("unknown module: %s", moduleName)
		}

		module, err := factory(ctx, log, cfg)
		if err != nil {
			return fmt.Errorf("failed to create module %s: %w", moduleName, err)
		}

		if err := registry.Register(module); err != nil {
			return fmt.Errorf("failed to register module %s: %w", moduleName, err)
		}
	}

	return nil
}
