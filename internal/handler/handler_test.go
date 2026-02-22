package handler

import (
	"context"
	"testing"
	"time"

	"github.com/minimax-agent/telegram-bot/internal/telegram"
)

func TestHandlerCommands(t *testing.T) {
	// Test command registration
	h := &Handler{
		commands: make(map[string]CommandHandler),
	}

	// Register a test command
	h.RegisterCommand("test", func(ctx context.Context, msg *telegram.Message, args string) error {
		return nil
	})

	// Verify command is registered
	_, ok := h.commands["test"]
	if !ok {
		t.Error("Command should be registered")
	}

	// Verify case insensitivity
	h.RegisterCommand("UPPERCASE", func(ctx context.Context, msg *telegram.Message, args string) error {
		return nil
	})
	_, ok = h.commands["uppercase"]
	if !ok {
		t.Error("Command should be registered in lowercase")
	}
}

func TestRateLimiting(t *testing.T) {
	h := &Handler{
		rateLimit:       1 * time.Second,
		lastMessageTime: make(map[int64]time.Time),
	}

	// First message should pass
	canSend := h.checkRateLimit(123)
	if !canSend {
		t.Error("First message should pass rate limit")
	}

	// Simulate message sent
	h.updateRateLimit(123)

	// Immediate second message should fail
	canSend = h.checkRateLimit(123)
	if canSend {
		t.Error("Second message should fail rate limit")
	}
}

func TestIsProcessing(t *testing.T) {
	h := &Handler{
		processing: make(map[int64]bool),
	}

	// Initially not processing
	isProc := h.isProcessing(123)
	if isProc {
		t.Error("Should not be processing initially")
	}

	// Set as processing
	h.setProcessing(123, true)

	// Now should be processing
	isProc = h.isProcessing(123)
	if !isProc {
		t.Error("Should be processing")
	}

	// Set as not processing
	h.setProcessing(123, false)

	// Now should not be processing
	isProc = h.isProcessing(123)
	if isProc {
		t.Error("Should not be processing")
	}
}
