// Package config provides configuration management for the Telegram bot.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration settings for the bot.
type Config struct {
	// Telegram Bot Configuration
	TelegramBotToken string `mapstructure:"telegram_bot_token"`

	// Minimax API Configuration
	MinimaxAPIKey  string        `mapstructure:"minimax_api_key"`
	MinimaxBaseURL string        `mapstructure:"minimax_base_url"`
	MinimaxModel   string        `mapstructure:"minimax_model"`
	MinimaxTimeout time.Duration `mapstructure:"minimax_timeout"`

	// Bot Configuration
	BotName         string  `mapstructure:"bot_name"`
	AdminUserIDs    []int64 `mapstructure:"admin_user_ids"`
	AllowedUsers    []int64 `mapstructure:"allowed_users"`
	EnableGroupChat bool    `mapstructure:"enable_group_chat"`

	// Polling Configuration
	PollInterval time.Duration `mapstructure:"poll_interval"`
	LongPolling  bool          `mapstructure:"long_polling"`

	// Message Configuration
	MaxMessageLength int           `mapstructure:"max_message_length"`
	ReplyTimeout     time.Duration `mapstructure:"reply_timeout"`

	// Feature Flags
	EnableMarkdown   bool `mapstructure:"enable_markdown"`
	EnableCommands   bool `mapstructure:"enable_commands"`
	EnableInlineMode bool `mapstructure:"enable_inline_mode"`
}

// Default returns a Config with default values.
func Default() *Config {
	return &Config{
		MinimaxBaseURL:   "https://api.minimax.chat/v1",
		MinimaxModel:     "abab5.5-chat",
		MinimaxTimeout:   60 * time.Second,
		PollInterval:     1 * time.Second,
		LongPolling:      true,
		MaxMessageLength: 4096,
		ReplyTimeout:     30 * time.Second,
		EnableMarkdown:   true,
		EnableCommands:   true,
		EnableInlineMode: false,
		EnableGroupChat:  false,
	}
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.TelegramBotToken == "" {
		return fmt.Errorf("telegram bot token is required")
	}

	if c.MinimaxAPIKey == "" {
		return fmt.Errorf("minimax api key is required")
	}

	if c.MinimaxBaseURL == "" {
		return fmt.Errorf("minimax base url is required")
	}

	if c.PollInterval <= 0 {
		c.PollInterval = 1 * time.Second
	}

	if c.MinimaxTimeout <= 0 {
		c.MinimaxTimeout = 60 * time.Second
	}

	return nil
}

// LoadFromEnv loads configuration from environment variables.
func LoadFromEnv() *Config {
	cfg := Default()

	// Telegram configuration
	if token := os.Getenv("TELEGRAM_BOT_TOKEN"); token != "" {
		cfg.TelegramBotToken = token
	}

	// Minimax configuration
	if apiKey := os.Getenv("MINIMAX_API_KEY"); apiKey != "" {
		cfg.MinimaxAPIKey = apiKey
	}

	if baseURL := os.Getenv("MINIMAX_BASE_URL"); baseURL != "" {
		cfg.MinimaxBaseURL = baseURL
	}

	if model := os.Getenv("MINIMAX_MODEL"); model != "" {
		cfg.MinimaxModel = model
	}

	// Bot configuration
	if name := os.Getenv("BOT_NAME"); name != "" {
		cfg.BotName = name
	}

	if adminIDs := os.Getenv("ADMIN_USER_IDS"); adminIDs != "" {
		cfg.AdminUserIDs = parseInt64List(adminIDs)
	}

	if allowedUsers := os.Getenv("ALLOWED_USERS"); allowedUsers != "" {
		cfg.AllowedUsers = parseInt64List(allowedUsers)
	}

	// Feature flags
	if enableGroup := os.Getenv("ENABLE_GROUP_CHAT"); enableGroup != "" {
		cfg.EnableGroupChat = enableGroup == "true" || enableGroup == "1"
	}

	if enableMarkdown := os.Getenv("ENABLE_MARKDOWN"); enableMarkdown != "" {
		cfg.EnableMarkdown = enableMarkdown == "true" || enableMarkdown == "1"
	}

	// Timeouts
	if pollInterval := os.Getenv("POLL_INTERVAL"); pollInterval != "" {
		if duration, err := time.ParseDuration(pollInterval); err == nil {
			cfg.PollInterval = duration
		}
	}

	if timeout := os.Getenv("MINIMAX_TIMEOUT"); timeout != "" {
		if duration, err := time.ParseDuration(timeout); err == nil {
			cfg.MinimaxTimeout = duration
		}
	}

	return cfg
}

// parseInt64List parses a comma-separated string of integers into a slice.
func parseInt64List(s string) []int64 {
	if s == "" {
		return nil
	}

	var result []int64
	for _, part := range splitAndTrim(s, ",") {
		if id, err := strconv.ParseInt(part, 10, 64); err == nil {
			result = append(result, id)
		}
	}
	return result
}

// splitAndTrim splits a string by separator and trims whitespace from each part.
func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, p := range split(s, sep) {
		if trimmed := trim(p); trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func split(s, sep string) []string {
	if s == "" {
		return nil
	}
	result := make([]string, 0)
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
