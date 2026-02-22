package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestLoggerOptions(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping logger test in short mode")
	}
	buf := &bytes.Buffer{}

	// Test WithOutput option
	l := New(
		WithOutput(buf),
		WithLevel(DebugLevel),
		WithPrefix("test"),
	)

	l.Debug("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain 'test message', got %s", output)
	}
}

func TestLogLevels(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping logger test in short mode")
	}
	// Test basic logging works
	buf := &bytes.Buffer{}
	l := New(
		WithOutput(buf),
		WithLevel(ErrorLevel),
	)

	l.Error("error message")

	output := buf.String()
	if !strings.Contains(output, "error message") {
		t.Errorf("Expected output to contain 'error message', got %s", output)
	}
}

func TestLoggerWith(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping logger test in short mode")
	}
	// Basic test for With method
	buf := &bytes.Buffer{}
	l := New(
		WithOutput(buf),
	)

	child := l.With("child")
	child.Info("test")

	output := buf.String()
	if !strings.Contains(output, "child") {
		t.Errorf("Expected output to contain 'child', got %s", output)
	}
}

func TestPackageLevelLogger(t *testing.T) {
	// Save original logger
	origLogger := defaultLogger

	// Replace with test logger
	buf := &bytes.Buffer{}
	defaultLogger = New(
		WithOutput(buf),
		WithLevel(DebugLevel),
	)

	// Test package-level functions
	Debug("debug test")
	Info("info test")
	Warn("warn test")
	Error("error test")

	output := buf.String()

	if !strings.Contains(output, "debug test") {
		t.Errorf("Expected output to contain 'debug test', got %s", output)
	}

	if !strings.Contains(output, "info test") {
		t.Errorf("Expected output to contain 'info test', got %s", output)
	}

	if !strings.Contains(output, "warn test") {
		t.Errorf("Expected output to contain 'warn test', got %s", output)
	}

	if !strings.Contains(output, "error test") {
		t.Errorf("Expected output to contain 'error test', got %s", output)
	}

	// Restore original logger
	defaultLogger = origLogger
}

func TestSetLevel(t *testing.T) {
	buf := &bytes.Buffer{}
	l := New(
		WithOutput(buf),
		WithLevel(ErrorLevel),
	)

	// Debug should not be logged
	l.Debug("this should not appear")

	output := buf.String()
	if strings.Contains(output, "this should not appear") {
		t.Error("Debug message should not be logged when level is Error")
	}

	// Error should be logged
	l.Error("this should appear")

	output = buf.String()
	if !strings.Contains(output, "this should appear") {
		t.Error("Error message should be logged")
	}
}

func TestLoggerLevels(t *testing.T) {
	if DebugLevel != 0 {
		t.Errorf("Expected DebugLevel to be 0, got %d", DebugLevel)
	}

	if InfoLevel != 1 {
		t.Errorf("Expected InfoLevel to be 1, got %d", InfoLevel)
	}

	if WarnLevel != 2 {
		t.Errorf("Expected WarnLevel to be 2, got %d", WarnLevel)
	}

	if ErrorLevel != 3 {
		t.Errorf("Expected ErrorLevel to be 3, got %d", ErrorLevel)
	}

	if FatalLevel != 4 {
		t.Errorf("Expected FatalLevel to be 4, got %d", FatalLevel)
	}
}
