# Prompt: Debug and Refactor Go Background Task Processing System

The Go background task processing application we've created has some issues and isn't functioning properly. Help identify and fix these problems by implementing comprehensive debug logging and refactoring the codebase to improve reliability and performance.

## Supporting Information

When we run the app - tasks are not getting completed. The metrics show that tasks are being queued and processed, but the TasksCompleted count is lower than expected.


## Steps to Complete

1. Add comprehensive logging to examples/task.go
   - Pass logger instance into the TaskGenerator and all task types
   - Log task creation, execution start, progress, and completion
   - Add task ID and name to all log entries
   - Log duration of sleep periods and execution times

2. Modify the TaskGenerator in examples/task.go
   - Reduce the MinSleep and MaxSleep values to make tasks complete faster
   - Current values (MinSleep: 100ms, MaxSleep: 2s) are too high
   - Consider reducing MinSleep to 10ms and MaxSleep to 200ms

3. Debug task completion issues
   - Add logging to verify tasks are properly completing their Execute method
   - Check for potential context cancellation issues
   - Verify the worker pool is properly tracking task completion

4. Refactor the main function
   - Break down into smaller, focused functions (initConfig, setupSignalHandling, createAndStartPool, etc.)
   - Move metric display to a separate function
   - Improve readability and maintainability
   - Ensure proper separation of concerns

---
_Filename: prompt-2-debug-and-refactor-go-background-task-processing-system.md_