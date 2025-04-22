# Prompt: Restructure Go Background Task Application with Modular Architecture

The Go background task application has been improved and now tasks are completing correctly. The codebase needs restructuring into a modular architecture where components can operate independently and potentially be moved to separate services in the future.

## Supporting Information

- The application should support multiple modules that can be enabled/disabled
- The background worker system should be one such module
- Modules should communicate via channels for now, but be designed for future separation
- Configuration should be environment variable based
- The current AppContext should be converted into a module

## Steps to Complete

1. Create a Module Framework
   - Define a core `Module` interface:
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
   - Create a module registry for registration and dependency management
   - Implement module lifecycle management in the application

2. Convert Background Worker to a Module
   - Rename to HephaestusModule (Greek god of craftsmanship and automation)
   - Move current worker pool, dispatcher, and related code to this module
   - Expose a task submission channel as the module's API
   - Make the module properly implement the Module interface
   - Handle internal errors without crashing the entire application

3. Implement environment variable-based configuration management
   - Create a dedicated config package
   - Support module-specific configuration via environment variables using the following naming pattern:
     - `CHASQUI_MODULES` - Comma-separated list of modules to enable
     - `CHASQUI_HEPHAESTUS_WORKERS` - Number of workers in the pool
     - `CHASQUI_HEPHAESTUS_QUEUE_SIZE` - Size of the task queue
     - `CHASQUI_HEPHAESTUS_BATCH_SIZE` - Batch size for the dispatcher
     - `CHASQUI_HEPHAESTUS_SHUTDOWN_WAIT` - Wait time for graceful shutdown (in seconds)
     - `CHASQUI_GENERATOR_COUNT` - Number of tasks to generate
     - `CHASQUI_GENERATOR_INTERVAL` - Interval between task generation
   - Create a .env.example file with all configurable options and documentation
   - Add .env to .gitignore
   - Update the README with configuration instructions and variable descriptions
   - Update the README to explain the naming: "Hephaestus - the Greek god of craftsmanship and automation - is the module responsible for processing background tasks"
   - Provide sensible defaults for all configuration options
   - Add validation for configuration values

4. Convert example code into a separate TaskGeneratorModule
   - Move task generation logic to its own module
   - Connect to the worker module using channels
   - Make it configurable and independently testable

5. Update main.go to use the module system
   - Keep main.go small and focused on application bootstrapping
   - Load configuration and determine which modules to start
   - Manage overall application lifecycle
   - Properly handle signals for graceful shutdown

---
_Filename: prompt-3-restructure-go-background-task-application-with-modular-architecture.md_