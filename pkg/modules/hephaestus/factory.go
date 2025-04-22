package hephaestus

import (
	"context"

	"github.com/user/chasqui/pkg/config"
	"github.com/user/chasqui/pkg/logger"
	"github.com/user/chasqui/pkg/module"
)

// Factory creates a new Hephaestus module from application configuration
func Factory(ctx context.Context, log *logger.Logger, appConfig *config.Config) (module.Module, error) {
	// Create module-specific configuration
	moduleConfig := Config{
		Workers:   appConfig.Hephaestus.Workers,
		QueueSize: appConfig.Hephaestus.QueueSize,
		BatchSize: appConfig.Hephaestus.BatchSize,
		ShutdownTimeout: module.Duration{
			Duration: appConfig.Hephaestus.ShutdownWait,
		},
	}

	// Create and return the module
	return NewModule(ctx, log, moduleConfig)
}
