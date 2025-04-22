package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	// Environment variable prefixes
	EnvPrefix = "CHASQUI_"

	// Common environment variables
	EnvModules = EnvPrefix + "MODULES"

	// Default values
	DefaultModules = "hephaestus,generator"
)

// Config holds the application configuration
type Config struct {
	// General configuration
	EnabledModules []string

	// Module-specific configurations
	Hephaestus HephaestusConfig
	Generator  GeneratorConfig
}

// HephaestusConfig holds configuration for the Hephaestus module
type HephaestusConfig struct {
	Workers      int
	QueueSize    int
	BatchSize    int
	ShutdownWait time.Duration
}

// GeneratorConfig holds configuration for the task generator module
type GeneratorConfig struct {
	Count    int
	Interval time.Duration
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	config := &Config{
		EnabledModules: getStringSlice(EnvModules, DefaultModules),
		Hephaestus: HephaestusConfig{
			Workers:      getInt(EnvPrefix+"HEPHAESTUS_WORKERS", 5),
			QueueSize:    getInt(EnvPrefix+"HEPHAESTUS_QUEUE_SIZE", 100),
			BatchSize:    getInt(EnvPrefix+"HEPHAESTUS_BATCH_SIZE", 10),
			ShutdownWait: getDuration(EnvPrefix+"HEPHAESTUS_SHUTDOWN_WAIT", 10*time.Second),
		},
		Generator: GeneratorConfig{
			Count:    getInt(EnvPrefix+"GENERATOR_COUNT", 50),
			Interval: getDuration(EnvPrefix+"GENERATOR_INTERVAL", 50*time.Millisecond),
		},
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
}

// validateConfig validates the configuration values
func validateConfig(cfg *Config) error {
	// Validate Hephaestus config
	if cfg.Hephaestus.Workers <= 0 {
		return fmt.Errorf("HEPHAESTUS_WORKERS must be greater than 0")
	}
	if cfg.Hephaestus.QueueSize <= 0 {
		return fmt.Errorf("HEPHAESTUS_QUEUE_SIZE must be greater than 0")
	}
	if cfg.Hephaestus.BatchSize <= 0 {
		return fmt.Errorf("HEPHAESTUS_BATCH_SIZE must be greater than 0")
	}
	if cfg.Hephaestus.ShutdownWait < time.Second {
		return fmt.Errorf("HEPHAESTUS_SHUTDOWN_WAIT must be at least 1 second")
	}

	// Validate Generator config
	if cfg.Generator.Count < 0 {
		return fmt.Errorf("GENERATOR_COUNT must be non-negative")
	}
	if cfg.Generator.Interval < time.Millisecond {
		return fmt.Errorf("GENERATOR_INTERVAL must be at least 1 millisecond")
	}

	return nil
}

// Helper functions to get configuration values from environment variables

// getInt reads an integer from an environment variable with a default value
func getInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		fmt.Printf("Warning: Invalid value for %s: %s. Using default: %d\n", key, value, defaultValue)
		return defaultValue
	}

	return intValue
}

// getStringSlice reads a comma-separated list from an environment variable with a default value
func getStringSlice(key string, defaultValue string) []string {
	value := os.Getenv(key)
	if value == "" {
		value = defaultValue
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// getDuration reads a duration from an environment variable with a default value
func getDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	// Try to parse as seconds first
	seconds, err := strconv.Atoi(value)
	if err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try to parse as a duration string
	duration, err := time.ParseDuration(value)
	if err != nil {
		fmt.Printf("Warning: Invalid duration for %s: %s. Using default: %v\n", key, value, defaultValue)
		return defaultValue
	}

	return duration
}

// PrintConfig prints the current configuration
func PrintConfig(cfg *Config) {
	fmt.Println("=== Chasqui Configuration ===")
	fmt.Printf("Enabled Modules: %s\n", strings.Join(cfg.EnabledModules, ", "))

	fmt.Println("\nHephaestus Module:")
	fmt.Printf("  Workers:       %d\n", cfg.Hephaestus.Workers)
	fmt.Printf("  Queue Size:    %d\n", cfg.Hephaestus.QueueSize)
	fmt.Printf("  Batch Size:    %d\n", cfg.Hephaestus.BatchSize)
	fmt.Printf("  Shutdown Wait: %v\n", cfg.Hephaestus.ShutdownWait)

	fmt.Println("\nGenerator Module:")
	fmt.Printf("  Count:    %d\n", cfg.Generator.Count)
	fmt.Printf("  Interval: %v\n", cfg.Generator.Interval)
}
