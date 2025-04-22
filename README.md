# Chasqui: Concurrent Background Task Processing in Go

Chasqui is a Go library for executing multiple background tasks concurrently using a worker pool pattern with a modular architecture. The library provides a simple, efficient way to process tasks asynchronously with proper error handling, logging, and graceful shutdown capabilities.

## Features

- **Modular Architecture**: Independent modules that can be enabled or disabled
- **Worker Pool**: Manages a configurable number of goroutines to process tasks concurrently (Hephaestus module)
- **Task Interface**: Common `Work` interface for implementing diverse task types
- **Dispatcher**: Batches and schedules tasks efficiently
- **Graceful Shutdown**: Proper handling of in-progress tasks during shutdown
- **Error Handling**: Robust error reporting and recovery
- **Logging**: Comprehensive logging of task lifecycle events
- **Backpressure Handling**: Mechanisms to handle queue overflow
- **Metrics**: Basic task processing metrics

## Installation

```bash
go get github.com/user/chasqui
```

## Module System

Chasqui uses a modular architecture where components can operate independently and potentially be moved to separate services in the future.

### Available Modules

1. **Hephaestus Module** - The Greek god of craftsmanship and automation - is the module responsible for processing background tasks. It provides:
   - Worker pool for concurrent task execution
   - Task dispatcher for batching
   - Metrics for monitoring

2. **Generator Module** - Creates example tasks and submits them to the Hephaestus module for processing.

### Module Interface

All modules implement the following interface:

```go
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
```

## Configuration

Chasqui uses environment variables for configuration. The following variables are available:

### General Configuration

- `CHASQUI_MODULES` - Comma-separated list of modules to enable (default: "hephaestus,generator")

### Hephaestus Module Configuration

- `CHASQUI_HEPHAESTUS_WORKERS` - Number of workers in the pool (default: 5)
- `CHASQUI_HEPHAESTUS_QUEUE_SIZE` - Size of the task queue (default: 100)
- `CHASQUI_HEPHAESTUS_BATCH_SIZE` - Batch size for the dispatcher (default: 10)
- `CHASQUI_HEPHAESTUS_SHUTDOWN_WAIT` - Wait time for graceful shutdown in seconds (default: 10)

### Generator Module Configuration

- `CHASQUI_GENERATOR_COUNT` - Number of tasks to generate (default: 50)
- `CHASQUI_GENERATOR_INTERVAL` - Interval between task generation (default: 50ms)

See [.env.example](.env.example) for a complete list of configuration options.

## Quick Start

1. Clone the repository
2. Create a `.env` file with your configuration (or use the defaults)
3. Build and run:

```bash
go build -o chasqui ./cmd/chasqui
./chasqui
```

## Development

To extend Chasqui with new modules:

1. Create a new package in `pkg/modules/<your-module-name>`
2. Implement the `module.Module` interface
3. Create a factory function to initialize your module from configuration
4. Register your module in `main.go`

## License

This project is licensed under the MIT License - see the LICENSE file for details. 