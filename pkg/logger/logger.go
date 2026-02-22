// Package logger provides structured logging for the bot.
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// Level represents the severity of a log message.
type Level int

const (
	// DebugLevel is for detailed diagnostic information.
	DebugLevel Level = iota
	// InfoLevel is for general informational messages.
	InfoLevel
	// WarnLevel is for warning messages about potential issues.
	WarnLevel
	// ErrorLevel is for error messages.
	ErrorLevel
	// FatalLevel is for critical errors that cause the program to exit.
	FatalLevel
)

var levelNames = map[Level]string{
	DebugLevel: "DEBUG",
	InfoLevel:  "INFO",
	WarnLevel:  "WARN",
	ErrorLevel: "ERROR",
	FatalLevel: "FATAL",
}

// Logger provides structured logging capabilities.
type Logger struct {
	mu         sync.Mutex
	output     io.Writer
	prefix     string
	level      Level
	flags      int
	timeFormat string
}

// Option configures a Logger.
type Option func(*Logger)

// WithOutput sets the output writer for the logger.
func WithOutput(w io.Writer) Option {
	return func(l *Logger) {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.output = w
	}
}

// WithPrefix sets a prefix for log messages.
func WithPrefix(prefix string) Option {
	return func(l *Logger) {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.prefix = prefix
	}
}

// WithLevel sets the minimum log level.
func WithLevel(level Level) Option {
	return func(l *Logger) {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.level = level
	}
}

// WithTimeFormat sets the time format for log messages.
func WithTimeFormat(format string) Option {
	return func(l *Logger) {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.timeFormat = format
	}
}

// New creates a new Logger with the given options.
func New(opts ...Option) *Logger {
	logger := &Logger{
		output:     os.Stderr,
		level:      InfoLevel,
		flags:      log.LstdFlags,
		timeFormat: time.RFC3339,
	}

	for _, opt := range opts {
		opt(logger)
	}

	return logger
}

// Default returns a logger configured with sensible defaults.
func Default() *Logger {
	return New(
		WithLevel(InfoLevel),
		WithOutput(os.Stderr),
	)
}

// log writes a log message at the specified level.
func (l *Logger) log(level Level, format string, args ...interface{}) {
	if level < l.level {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format(l.timeFormat)
	levelName := levelNames[level]
	message := fmt.Sprintf(format, args...)

	if l.prefix != "" {
		fmt.Fprintf(l.output, "[%s] [%s] %s: %s\n", timestamp, levelName, l.prefix, message)
	} else {
		fmt.Fprintf(l.output, "[%s] [%s] %s\n", timestamp, levelName, message)
	}
}

// Debug logs a message at debug level.
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DebugLevel, format, args...)
}

// Info logs a message at info level.
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(InfoLevel, format, args...)
}

// Warn logs a message at warning level.
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WarnLevel, format, args...)
}

// Error logs a message at error level.
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ErrorLevel, format, args...)
}

// Fatal logs a message at fatal level and exits the program.
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(FatalLevel, format, args...)
	os.Exit(1)
}

// With creates a child logger with the given prefix.
func (l *Logger) With(prefix string) *Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	child := &Logger{
		output:     l.output,
		prefix:     l.prefix + "/" + prefix,
		level:      l.level,
		flags:      l.flags,
		timeFormat: l.timeFormat,
	}

	return child
}

// Package-level logger instance
var defaultLogger = Default()

// Debug logs a message at debug level using the default logger.
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

// Info logs a message at info level using the default logger.
func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

// Warn logs a message at warning level using the default logger.
func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

// Error logs a message at error level using the default logger.
func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

// Fatal logs a message at fatal level and exits the program using the default logger.
func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}

// SetLevel sets the minimum log level for the default logger.
func SetLevel(level Level) {
	defaultLogger.level = level
}
