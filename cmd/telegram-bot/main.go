// Command line interface for the Telegram Minimax Bot.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/minimax-agent/telegram-bot/internal/handler"
	"github.com/minimax-agent/telegram-bot/internal/minimax"
	"github.com/minimax-agent/telegram-bot/internal/telegram"
	"github.com/minimax-agent/telegram-bot/pkg/config"
	"github.com/minimax-agent/telegram-bot/pkg/logger"
)

const (
	appName    = "Telegram Minimax Bot"
	appVersion = "1.0.0"
)

func main() {
	// Load .env file if it exists (doesn't error if missing)
	godotenv.Load()

	// Load configuration
	cfg := config.LoadFromEnv()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	// Setup logger
	log := logger.New(
		logger.WithPrefix("bot"),
		logger.WithLevel(logger.InfoLevel),
	)

	log.Info("Starting %s v%s", appName, appVersion)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create Telegram client
	telegramClient, err := telegram.NewClient(
		cfg.TelegramBotToken,
		telegram.WithLogger(log),
	)
	if err != nil {
		log.Fatal("Failed to create Telegram client: %v", err)
	}

	// Get bot info
	botInfo, err := telegramClient.GetMe(ctx)
	if err != nil {
		log.Fatal("Failed to get bot info: %v", err)
	}
	log.Info("Logged in as @%s (ID: %d)", botInfo.Username, botInfo.ID)

	// Create Minimax client
	minimaxClient, err := minimax.NewClient(
		cfg.MinimaxAPIKey,
		minimax.WithBaseURL(cfg.MinimaxBaseURL),
		minimax.WithModel(cfg.MinimaxModel),
		minimax.WithTimeout(cfg.MinimaxTimeout),
		minimax.WithLogger(log),
	)
	if err != nil {
		log.Fatal("Failed to create Minimax client: %v", err)
	}
	log.Info("Minimax client initialized with model: %s", cfg.MinimaxModel)

	// Create handler
	h := handler.New(telegramClient, minimaxClient, cfg)

	// Start long polling
	log.Info("Starting long polling...")
	err = telegramClient.StartLongPolling(ctx)
	if err != nil {
		log.Fatal("Failed to start long polling: %v", err)
	}

	// Handle updates in a goroutine
	go func() {
		updateCh := telegramClient.GetUpdateChannel()
		for {
			select {
			case <-ctx.Done():
				return
			case update, ok := <-updateCh:
				if !ok {
					return
				}
				if err := h.HandleUpdate(ctx, update); err != nil {
					log.Error("Error handling update: %v", err)
				}
			}
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info("Shutting down...")

	// Stop long polling
	telegramClient.StopLongPolling()

	// Cancel context
	cancel()

	// Give time for graceful shutdown
	time.Sleep(time.Second)

	log.Info("Bot stopped")
}
