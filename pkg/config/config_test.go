package config

import (
	"os"
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.MinimaxBaseURL != "https://api.minimax.chat/v1" {
		t.Errorf("Expected default MinimaxBaseURL, got %s", cfg.MinimaxBaseURL)
	}

	if cfg.MinimaxModel != "abab5.5-chat" {
		t.Errorf("Expected default MinimaxModel, got %s", cfg.MinimaxModel)
	}

	if cfg.PollInterval != 1*time.Second {
		t.Errorf("Expected default PollInterval, got %v", cfg.PollInterval)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: &Config{
				TelegramBotToken: "test_token",
				MinimaxAPIKey:    "test_key",
				MinimaxBaseURL:   "https://api.minimax.chat/v1",
			},
			wantErr: false,
		},
		{
			name: "missing telegram token",
			cfg: &Config{
				TelegramBotToken: "",
				MinimaxAPIKey:    "test_key",
				MinimaxBaseURL:   "https://api.minimax.chat/v1",
			},
			wantErr: true,
		},
		{
			name: "missing minimax api key",
			cfg: &Config{
				TelegramBotToken: "test_token",
				MinimaxAPIKey:    "",
				MinimaxBaseURL:   "https://api.minimax.chat/v1",
			},
			wantErr: true,
		},
		{
			name: "missing minimax base url",
			cfg: &Config{
				TelegramBotToken: "test_token",
				MinimaxAPIKey:    "test_key",
				MinimaxBaseURL:   "",
			},
			wantErr: true,
		},
		{
			name: "zero poll interval defaults to 1s",
			cfg: &Config{
				TelegramBotToken: "test_token",
				MinimaxAPIKey:    "test_key",
				MinimaxBaseURL:   "https://api.minimax.chat/v1",
				PollInterval:     0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Clear any existing env vars
	os.Unsetenv("TELEGRAM_BOT_TOKEN")
	os.Unsetenv("MINIMAX_API_KEY")
	os.Unsetenv("MINIMAX_BASE_URL")
	os.Unsetenv("MINIMAX_MODEL")
	os.Unsetenv("BOT_NAME")

	// Test with no env vars
	cfg := LoadFromEnv()
	if cfg.TelegramBotToken != "" {
		t.Errorf("Expected empty TelegramBotToken, got %s", cfg.TelegramBotToken)
	}

	// Set env vars
	os.Setenv("TELEGRAM_BOT_TOKEN", "test_token_123")
	os.Setenv("MINIMAX_API_KEY", "minimax_key_456")
	os.Setenv("MINIMAX_BASE_URL", "https://custom.api.minimax.chat/v1")
	os.Setenv("MINIMAX_MODEL", "custom-model")
	os.Setenv("BOT_NAME", "my-test-bot")

	// Reload
	cfg = LoadFromEnv()

	if cfg.TelegramBotToken != "test_token_123" {
		t.Errorf("Expected TELEGRAM_BOT_TOKEN, got %s", cfg.TelegramBotToken)
	}

	if cfg.MinimaxAPIKey != "minimax_key_456" {
		t.Errorf("Expected MINIMAX_API_KEY, got %s", cfg.MinimaxAPIKey)
	}

	if cfg.MinimaxBaseURL != "https://custom.api.minimax.chat/v1" {
		t.Errorf("Expected custom MINIMAX_BASE_URL, got %s", cfg.MinimaxBaseURL)
	}

	if cfg.MinimaxModel != "custom-model" {
		t.Errorf("Expected MINIMAX_MODEL, got %s", cfg.MinimaxModel)
	}

	if cfg.BotName != "my-test-bot" {
		t.Errorf("Expected BOT_NAME, got %s", cfg.BotName)
	}

	// Clean up
	os.Unsetenv("TELEGRAM_BOT_TOKEN")
	os.Unsetenv("MINIMAX_API_KEY")
	os.Unsetenv("MINIMAX_BASE_URL")
	os.Unsetenv("MINIMAX_MODEL")
	os.Unsetenv("BOT_NAME")
}

func TestParseInt64List(t *testing.T) {
	tests := []struct {
		input string
		want  []int64
	}{
		{"123,456,789", []int64{123, 456, 789}},
		{" 123 , 456 , 789 ", []int64{123, 456, 789}},
		{"", nil},
		{"abc", nil},
		{"123,abc,456", []int64{123, 456}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseInt64List(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseInt64List(%q) = %v, want %v", tt.input, got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseInt64List(%q)[%d] = %v, want %v", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}
