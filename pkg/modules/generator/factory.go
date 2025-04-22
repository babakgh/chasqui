package generator

import (
	"context"
	"time"

	"github.com/user/chasqui/pkg/config"
	"github.com/user/chasqui/pkg/logger"
	"github.com/user/chasqui/pkg/module"
)

// Factory creates a new Generator module from application configuration
func Factory(ctx context.Context, log *logger.Logger, appConfig *config.Config) (module.Module, error) {
	// Create module-specific configuration
	moduleConfig := Config{
		Count:       appConfig.Generator.Count,
		Interval:    appConfig.Generator.Interval,
		MinDuration: 10 * time.Millisecond,  // Default minimum duration
		MaxDuration: 200 * time.Millisecond, // Default maximum duration
		FailProb:    0.2,                    // Default failure probability
	}

	// Create and return the module
	return NewModule(ctx, log, moduleConfig)
}
