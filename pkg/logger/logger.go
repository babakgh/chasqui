package logger

import (
	"log"
	"os"
	"time"
)

// LogLevel represents different levels of logging.
type LogLevel int

const (
	// DEBUG level for detailed information, typically of interest only when diagnosing problems.
	DEBUG LogLevel = iota
	// INFO level for confirmation that things are working as expected.
	INFO
	// WARN level for situations that might cause problems but are recoverable.
	WARN
	// ERROR level for error events that might still allow the application to continue running.
	ERROR
	// FATAL level for severe error events that will likely lead the application to abort.
	FATAL
)

// Logger provides logging functionality.
type Logger struct {
	debugLogger *log.Logger
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	fatalLogger *log.Logger
	level       LogLevel
}

// New creates a new Logger with the specified minimum log level.
func New(level LogLevel) *Logger {
	return &Logger{
		debugLogger: log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime),
		infoLogger:  log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime),
		warnLogger:  log.New(os.Stdout, "WARN: ", log.Ldate|log.Ltime),
		errorLogger: log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime),
		fatalLogger: log.New(os.Stderr, "FATAL: ", log.Ldate|log.Ltime),
		level:       level,
	}
}

// Debug logs a message at DEBUG level if the logger's level permits it.
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.level <= DEBUG {
		l.debugLogger.Printf(format, v...)
	}
}

// Info logs a message at INFO level if the logger's level permits it.
func (l *Logger) Info(format string, v ...interface{}) {
	if l.level <= INFO {
		l.infoLogger.Printf(format, v...)
	}
}

// Warn logs a message at WARN level if the logger's level permits it.
func (l *Logger) Warn(format string, v ...interface{}) {
	if l.level <= WARN {
		l.warnLogger.Printf(format, v...)
	}
}

// Error logs a message at ERROR level if the logger's level permits it.
func (l *Logger) Error(format string, v ...interface{}) {
	if l.level <= ERROR {
		l.errorLogger.Printf(format, v...)
	}
}

// Fatal logs a message at FATAL level and then calls os.Exit(1).
func (l *Logger) Fatal(format string, v ...interface{}) {
	if l.level <= FATAL {
		l.fatalLogger.Printf(format, v...)
		os.Exit(1)
	}
}

// LogFunc logs the duration and result of the given function.
func (l *Logger) LogFunc(name string, fn func() error) error {
	l.Info("Starting %s", name)
	start := time.Now()

	err := fn()

	duration := time.Since(start)
	if err != nil {
		l.Error("%s failed after %v: %v", name, duration, err)
	} else {
		l.Info("%s completed successfully in %v", name, duration)
	}

	return err
}
