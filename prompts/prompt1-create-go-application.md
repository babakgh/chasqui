# Prompt: Create a Go Application for Concurrent Background Task Processing

Create a Go application that can execute multiple background tasks concurrently using a worker pool pattern. The application should be able to run any task that implements a `Work` interface, and tasks will be fed to the system via a channel.

## Supporting Information

- The system needs a worker pool that manages a configurable number of goroutines
- Each worker should consume tasks from a shared channel
- Tasks must implement a common `Work` interface
- The application should gracefully handle task failures
- Include proper logging and error handling
- The system should be able to gracefully shut down when requested
- Consider resource utilization and backpressure handling
- Create a proper .gitignore file to exclude unnecessary files (binaries, IDE settings, temporary files, etc.)

## Steps to Complete

1. Set up the project structure and create a proper .gitignore file for Go projects
2. Define a `Work` interface that any task must implement
3. Create a worker pool structure that manages goroutines
4. Implement a task dispatcher that feeds tasks to the worker pool
5. Add proper error handling and logging mechanisms
6. Implement graceful shutdown functionality
7. Add configuration options for pool size, queue capacity, etc.
8. Create example tasks that implement the `Work` interface
9. Write tests to validate the system behavior
10. Add documentation with usage examples