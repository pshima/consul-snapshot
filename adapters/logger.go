package adapters

import (
	"log"

	"github.com/pshima/consul-snapshot/interfaces"
)

// LoggerAdapter implements the Logger interface
type LoggerAdapter struct{}

// NewLoggerAdapter creates a new logger adapter
func NewLoggerAdapter() interfaces.Logger {
	return &LoggerAdapter{}
}

// Printf logs with format
func (l *LoggerAdapter) Printf(format string, args ...interface{}) {
	log.Printf(format, args...)
}

// Print logs without format
func (l *LoggerAdapter) Print(args ...interface{}) {
	log.Print(args...)
}

// Fatalf logs with format and exits
func (l *LoggerAdapter) Fatalf(format string, args ...interface{}) {
	log.Fatalf(format, args...)
}

// Fatal logs and exits
func (l *LoggerAdapter) Fatal(args ...interface{}) {
	log.Fatal(args...)
}