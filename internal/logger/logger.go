// Package logger provides a simple logger for terragrunt-ls.
package logger

import (
	"io"
	"log/slog"
	"os"
)

// Logger is a wrapper around slog.Logger that provides additional methods
type Logger struct {
	*slog.Logger
	closer io.Closer
}

// NewLogger builds the standard logger for terragrunt-ls.
//
// When supplied with a filename, it'll create a new file and write logs to it.
// Otherwise, it'll write logs to stderr.
func NewLogger(filename string) *Logger {
	if filename == "" {
		handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		})
		logger := slog.New(handler)

		return &Logger{
			Logger: logger,
		}
	}

	const readWritePerm = 0666

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, readWritePerm)
	if err != nil {
		slog.Error("Failed to open log file", "error", err)
		os.Exit(1)
	}

	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	logger := slog.New(handler)

	return &Logger{
		Logger: logger,
		closer: file,
	}
}

// Close closes the logger
func (l *Logger) Close() error {
	if l.closer != nil {
		return l.closer.Close()
	}

	return nil
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...interface{}) {
	l.Logger.Debug(msg, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	l.Logger.Info(msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	l.Logger.Error(msg, args...)
}
