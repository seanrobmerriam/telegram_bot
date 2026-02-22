// Package handler provides message handling for the Telegram bot.
package handler

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/minimax-agent/telegram-bot/internal/minimax"
	"github.com/minimax-agent/telegram-bot/internal/telegram"
	"github.com/minimax-agent/telegram-bot/internal/wizard"
	"github.com/minimax-agent/telegram-bot/pkg/config"
	"github.com/minimax-agent/telegram-bot/pkg/logger"
)

// Handler handles incoming updates from Telegram and communicates with Minimax.
type Handler struct {
	telegramClient *telegram.Client
	minimaxClient  *minimax.Client
	config         *config.Config
	logger         *logger.Logger

	// User processing status
	processingMu sync.RWMutex
	processing   map[int64]bool

	// Rate limiting
	rateLimitMu     sync.RWMutex
	lastMessageTime map[int64]time.Time
	rateLimit       time.Duration

	// Command handlers
	commands map[string]CommandHandler

	// Wizard manager
	wizardManager *wizard.Manager
}

// CommandHandler is a function that handles a command.
type CommandHandler func(ctx context.Context, msg *telegram.Message, args string) error

// New creates a new Handler.
func New(
	telegramClient *telegram.Client,
	minimaxClient *minimax.Client,
	cfg *config.Config,
) *Handler {
	h := &Handler{
		telegramClient:  telegramClient,
		minimaxClient:   minimaxClient,
		config:          cfg,
		logger:          logger.Default(),
		processing:      make(map[int64]bool),
		lastMessageTime: make(map[int64]time.Time),
		rateLimit:       1 * time.Second, // Rate limit per user
		commands:        make(map[string]CommandHandler),
		wizardManager:   wizard.NewManager(10 * time.Minute),
	}

	// Register default commands
	h.registerDefaultCommands()

	return h
}

// HandleUpdate handles an incoming update from Telegram.
func (h *Handler) HandleUpdate(ctx context.Context, update telegram.Update) error {
	// Handle different update types
	switch {
	case update.Message != nil:
		return h.handleMessage(ctx, update.Message)
	case update.CallbackQuery != nil:
		return h.handleCallbackQuery(ctx, update.CallbackQuery)
	case update.InlineQuery != nil:
		return h.handleInlineQuery(ctx, update.InlineQuery)
	default:
		h.logger.Debug("Unhandled update type: %+v", update)
	}

	return nil
}

// handleMessage handles an incoming message.
func (h *Handler) handleMessage(ctx context.Context, msg *telegram.Message) error {
	if msg == nil || msg.Text == "" {
		return nil
	}

	// Check if it's a command
	if strings.HasPrefix(msg.Text, "/") {
		return h.handleCommand(ctx, msg)
	}

	// Check if user has active wizard session
	if wiz, ok := h.wizardManager.GetWizard(msg.From.ID); ok {
		return h.handleWizardMessage(ctx, msg, wiz)
	}

	// Check rate limiting
	if !h.checkRateLimit(msg.From.ID) {
		h.sendMessage(ctx, msg.Chat.ID, "Please wait a moment before sending another message.")
		return nil
	}

	// Check if user is already processing
	if h.isProcessing(msg.From.ID) {
		h.sendMessage(ctx, msg.Chat.ID, "I'm still processing your previous message. Please wait.")
		return nil
	}

	// Set user as processing
	h.setProcessing(msg.From.ID, true)
	defer h.setProcessing(msg.From.ID, false)

	// Add user message to conversation
	h.minimaxClient.Chat(ctx, minimax.ChatParams{
		UserID: msg.From.ID,
		Messages: []minimax.Message{
			{
				Role:    "user",
				Content: msg.Text,
			},
		},
	})

	// Send thinking indicator
	thinkingMsg, err := h.sendMessage(ctx, msg.Chat.ID, "🤔 Thinking...")
	if err != nil {
		h.logger.Error("Failed to send thinking message: %v", err)
	}

	// Get response from Minimax
	response, err := h.minimaxClient.Chat(ctx, minimax.ChatParams{
		UserID: msg.From.ID,
	})
	if err != nil {
		h.logger.Error("Minimax error: %v", err)

		// Delete thinking message
		if thinkingMsg != nil {
			h.telegramClient.DeleteMessage(ctx, msg.Chat.ID, thinkingMsg.MessageID)
		}

		h.sendMessage(ctx, msg.Chat.ID, fmt.Sprintf("Sorry, I encountered an error: %v", err))
		return err
	}

	// Delete thinking message
	if thinkingMsg != nil {
		h.telegramClient.DeleteMessage(ctx, msg.Chat.ID, thinkingMsg.MessageID)
	}

	// Send response
	if len(response.Choices) > 0 {
		responseText := response.Choices[0].Message.Content
		h.sendMessage(ctx, msg.Chat.ID, responseText)
	}

	return nil
}

// handleWizardMessage handles a message in an active wizard session.
func (h *Handler) handleWizardMessage(ctx context.Context, msg *telegram.Message, wiz *wizard.Wizard) error {
	// Get current question key
	key := wiz.GetCurrentKey()

	// Save the answer
	wiz.SetAnswer(key, msg.Text)

	// Check if wizard is complete
	if wiz.IsComplete() {
		// Build prompt from answers
		prompt := wiz.BuildPrompt()

		// Clear wizard session
		h.wizardManager.EndWizard(msg.From.ID)

		// Generate content
		h.sendMessage(ctx, msg.Chat.ID, "Generating content based on your answers...")

		response, err := h.minimaxClient.Chat(ctx, minimax.ChatParams{
			UserID: msg.From.ID,
			Messages: []minimax.Message{
				{Role: "user", Content: prompt},
			},
		})
		if err != nil {
			h.sendMessage(ctx, msg.Chat.ID, fmt.Sprintf("Error generating content: %v", err))
			return err
		}

		if len(response.Choices) > 0 {
			h.sendMessage(ctx, msg.Chat.ID, response.Choices[0].Message.Content)
		}
		return nil
	}

	// Get next question
	nextQuestion := wiz.GetCurrentQuestion()
	progress := wiz.GetProgress()
	h.sendMessage(ctx, msg.Chat.ID, fmt.Sprintf("Got it! %s\n\n%s\n\n(Type /cancel to cancel the wizard)", progress, nextQuestion))
	return nil
}

// handleCommand handles a command message.
func (h *Handler) handleCommand(ctx context.Context, msg *telegram.Message) error {
	// Remove the command prefix
	text := strings.TrimSpace(msg.Text)

	// Split command and arguments
	parts := strings.Fields(text[1:]) // Remove leading /
	if len(parts) == 0 {
		return nil
	}

	command := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = strings.Join(parts[1:], " ")
	}

	// Look up command handler
	handler, ok := h.commands[command]
	if !ok {
		h.sendMessage(ctx, msg.Chat.ID, fmt.Sprintf("Unknown command: /%s", command))
		return nil
	}

	return handler(ctx, msg, args)
}

// handleCallbackQuery handles a callback query.
func (h *Handler) handleCallbackQuery(ctx context.Context, query *telegram.CallbackQuery) error {
	// Answer the callback query
	h.telegramClient.AnswerCallbackQuery(ctx, telegram.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
	})

	// Handle based on callback data
	if query.Data == "" {
		return nil
	}

	// Process callback data (can be extended for more functionality)
	h.logger.Debug("Callback query: %s", query.Data)

	return nil
}

// handleInlineQuery handles an inline query.
func (h *Handler) handleInlineQuery(ctx context.Context, query *telegram.InlineQuery) error {
	// Process inline query if enabled
	if !h.config.EnableInlineMode {
		return nil
	}

	// This would require implementing the inline query response
	// For now, we'll just log it
	h.logger.Debug("Inline query from %s: %s", query.From.Username, query.Query)

	return nil
}

// escapeMarkdown escapes special characters for MarkdownV2.
func escapeMarkdown(text string) string {
	// Escape special characters for MarkdownV2
	re := regexp.MustCompile(`([\_*\[\]()~` + "`" + `>#+-|={}.!])`)
	return re.ReplaceAllString(text, "\\$1")
}

// sendMessage sends a message to a chat.
func (h *Handler) sendMessage(ctx context.Context, chatID int64, text string) (*telegram.Message, error) {
	// Truncate message if too long
	if len(text) > h.config.MaxMessageLength {
		text = text[:h.config.MaxMessageLength-3] + "..."
	}

	// Always send as plain text to avoid Markdown parsing issues
	// The AI responses often contain characters that conflict with MarkdownV2

	msg, err := h.telegramClient.SendMessage(ctx, telegram.SendMessageParams{
		ChatID:                chatID,
		Text:                  text,
		DisableWebPagePreview: true,
	})

	if err != nil {
		h.logger.Error("Failed to send message: %v", err)
		return nil, err
	}

	return msg, nil
}

// registerDefaultCommands registers the default command handlers.
func (h *Handler) registerDefaultCommands() {
	// /start command
	h.commands["start"] = func(ctx context.Context, msg *telegram.Message, args string) error {
		welcomeText := "Welcome to Minimax Bot!\n\nI'm an AI assistant powered by Minimax. You can talk to me by sending messages.\n\nAvailable commands:\n/start - Show this welcome message\n/clear - Clear conversation history\n/help - Show help information\n/status - Show bot status\n/create - Content creation wizard"
		h.sendMessage(ctx, msg.Chat.ID, welcomeText)
		return nil
	}

	// /help command
	h.commands["help"] = func(ctx context.Context, msg *telegram.Message, args string) error {
		helpText := "Help\n\nYou can communicate with me by sending messages. I'll respond using Minimax AI.\n\nCommands:\n/start - Start the bot\n/clear - Clear conversation history\n/help - Show this help message\n/status - Show bot status\n/create - Content creation wizard\n/cancel - Cancel active wizard\n\nTips:\n- Be specific in your questions\n- Provide context when needed\n- Use follow-up questions for more details"
		h.sendMessage(ctx, msg.Chat.ID, helpText)
		return nil
	}

	// /clear command
	h.commands["clear"] = func(ctx context.Context, msg *telegram.Message, args string) error {
		h.minimaxClient.ClearConversation(msg.From.ID)

		_, err := h.telegramClient.SendMessage(ctx, telegram.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "🗑️ Conversation history cleared!",
		})
		return err
	}

	// /status command
	h.commands["status"] = func(ctx context.Context, msg *telegram.Message, args string) error {
		botInfo, err := h.telegramClient.GetMe(ctx)
		if err != nil {
			return err
		}

		conversation := h.minimaxClient.GetConversation(msg.From.ID)
		msgCount := len(conversation)

		statusText := fmt.Sprintf("Bot Status\n\nBot: @%s\nModel: %s\nYour messages in this conversation: %d", botInfo.Username, h.config.MinimaxModel, msgCount)

		_, err = h.telegramClient.SendMessage(ctx, telegram.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   statusText,
		})
		return err
	}

	// /create command - starts content creation wizard
	h.commands["create"] = func(ctx context.Context, msg *telegram.Message, args string) error {
		// Parse content type and flags from args
		flags, contentType := wizard.ParseFlags(args)

		if contentType == "" {
			// Show help for /create command
			helpText := "Content Creation Wizard\n\nUse /create to start an interactive wizard for creating content.\n\nUsage:\n/create <type> [flags]\n\nContent Types:\n- marketing - Marketing copy\n- email     - Email content\n- report    - Business report\n- script    - Video/podcast script\n- whitepaper - Whitepaper\n- story     - Creative story\n- poem       - Poem\n\nFlags:\n-t <text>  - Quick prompt (bypasses wizard)\n-m <text>  - Message/instructions\n-s <style> - Writing style\n-q          - Quick mode (fewer questions)\n\nExamples:\n/create marketing\n/create email -t newsletter signup\n/create report -s formal\n/create story -q"
			h.sendMessage(ctx, msg.Chat.ID, helpText)
			return nil
		}

		// Start wizard session
		wiz := h.wizardManager.StartWizard(msg.From.ID, wizard.ContentType(contentType))

		// If quick mode with prompt, skip wizard
		if flags["t"] != "" {
			// Build prompt from flags and generate content directly
			prompt := flags["t"]
			if flags["m"] != "" {
				prompt = flags["m"] + ": " + prompt
			}
			if flags["s"] != "" {
				prompt += " (style: " + flags["s"] + ")"
			}

			h.sendMessage(ctx, msg.Chat.ID, "Generating content...")

			response, err := h.minimaxClient.Chat(ctx, minimax.ChatParams{
				UserID: msg.From.ID,
				Messages: []minimax.Message{
					{Role: "user", Content: prompt},
				},
			})
			if err != nil {
				h.sendMessage(ctx, msg.Chat.ID, fmt.Sprintf("Error: %v", err))
				return err
			}

			if len(response.Choices) > 0 {
				h.sendMessage(ctx, msg.Chat.ID, response.Choices[0].Message.Content)
			}
			return nil
		}

		// Get first question
		question := wiz.GetCurrentQuestion()
		h.sendMessage(ctx, msg.Chat.ID, fmt.Sprintf("Starting %s wizard! %s\n\n%s",
			contentType, wiz.GetProgress(), question))

		return nil
	}

	// /cancel command - cancel wizard
	h.commands["cancel"] = func(ctx context.Context, msg *telegram.Message, args string) error {
		h.wizardManager.CancelWizard(msg.From.ID)
		_, err := h.telegramClient.SendMessage(ctx, telegram.SendMessageParams{
			ChatID: msg.Chat.ID,
			Text:   "Wizard cancelled. Your session has been reset.",
		})
		return err
	}
}

// RegisterCommand registers a custom command handler.
func (h *Handler) RegisterCommand(name string, handler CommandHandler) {
	h.commands[strings.ToLower(name)] = handler
}

// checkRateLimit checks if the user is within rate limits.
func (h *Handler) checkRateLimit(userID int64) bool {
	h.rateLimitMu.RLock()
	defer h.rateLimitMu.RUnlock()

	lastTime, exists := h.lastMessageTime[userID]
	if !exists {
		return true
	}

	return time.Since(lastTime) >= h.rateLimit
}

// updateRateLimit updates the last message time for a user.
func (h *Handler) updateRateLimit(userID int64) {
	h.rateLimitMu.Lock()
	defer h.rateLimitMu.Unlock()
	h.lastMessageTime[userID] = time.Now()
}

// isProcessing checks if a user is currently processing a message.
func (h *Handler) isProcessing(userID int64) bool {
	h.processingMu.RLock()
	defer h.processingMu.RUnlock()
	return h.processing[userID]
}

// setProcessing sets the processing status for a user.
func (h *Handler) setProcessing(userID int64, processing bool) {
	h.processingMu.Lock()
	defer h.processingMu.Unlock()

	if h.processing == nil {
		h.processing = make(map[int64]bool)
	}
	h.processing[userID] = processing

	// Also update rate limit
	if !processing {
		h.rateLimitMu.Lock()
		if h.lastMessageTime == nil {
			h.lastMessageTime = make(map[int64]time.Time)
		}
		h.lastMessageTime[userID] = time.Now()
		h.rateLimitMu.Unlock()
	}
}
