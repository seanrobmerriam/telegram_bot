// Package minimax provides a client for interacting with the Minimax AI API.
package minimax

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/minimax-agent/telegram-bot/pkg/logger"
)

// Client represents a Minimax AI API client.
type Client struct {
	mu         sync.RWMutex
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	timeout    time.Duration
	debug      bool
	logger     *logger.Logger

	// Conversation history per user
	conversations map[int64]*Conversation
}

// NewClient creates a new Minimax API client.
func NewClient(apiKey string, opts ...ClientOption) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("api key is required")
	}

	client := &Client{
		apiKey:        apiKey,
		baseURL:       "https://api.minimax.chat/v1",
		model:         "abab5.5-chat",
		httpClient:    &http.Client{Timeout: 60 * time.Second},
		timeout:       60 * time.Second,
		logger:        logger.Default(),
		conversations: make(map[int64]*Conversation),
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// ClientOption configures the Minimax client.
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL for the Minimax API.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

// WithModel sets the model to use.
func WithModel(model string) ClientOption {
	return func(c *Client) {
		c.model = model
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
		c.httpClient.Timeout = timeout
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithDebug enables debug mode.
func WithDebug(debug bool) ClientOption {
	return func(c *Client) {
		c.debug = debug
	}
}

// WithLogger sets a custom logger.
func WithLogger(l *logger.Logger) ClientOption {
	return func(c *Client) {
		c.logger = l
	}
}

// Message represents a message in the conversation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Name    string `json:"name,omitempty"`
}

// Conversation represents a conversation with the AI.
type Conversation struct {
	mu       sync.RWMutex
	Messages []Message
	System   string
}

// AddMessage adds a message to the conversation history.
func (c *Conversation) AddMessage(role, content string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Messages = append(c.Messages, Message{
		Role:    role,
		Content: content,
	})
}

// GetMessages returns a copy of the conversation messages.
func (c *Conversation) GetMessages() []Message {
	c.mu.RLock()
	defer c.mu.RUnlock()
	messages := make([]Message, len(c.Messages))
	copy(messages, c.Messages)
	return messages
}

// Clear clears the conversation history.
func (c *Conversation) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Messages = nil
}

// SetSystem sets the system prompt for the conversation.
func (c *Conversation) SetSystem(system string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.System = system
}

// ChatRequest represents a chat completion request.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature,omitempty"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	TopP        float64   `json:"top_p,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
	Stop        []string  `json:"stop,omitempty"`
}

// ChatResponse represents a chat completion response.
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice represents a completion choice.
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage represents token usage information.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail represents error details.
type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// ChatParams contains parameters for the Chat method.
type ChatParams struct {
	// UserID is the Telegram user ID for conversation tracking
	UserID int64
	// Messages is the list of messages to send (overrides conversation history)
	Messages []Message
	// Temperature controls randomness (0-2)
	Temperature float64
	// MaxTokens limits the response length
	MaxTokens int
	// TopP controls nucleus sampling
	TopP float64
	// ClearConversation clears the conversation history before this request
	ClearConversation bool
	// SystemPrompt sets a custom system prompt
	SystemPrompt string
}

// Chat sends a chat completion request to Minimax.
func (c *Client) Chat(ctx context.Context, params ChatParams) (*ChatResponse, error) {
	// Build messages
	messages := params.Messages

	// If no messages provided, use conversation history
	if messages == nil {
		conv := c.getOrCreateConversation(params.UserID)

		// Apply system prompt if provided
		if params.SystemPrompt != "" {
			conv.SetSystem(params.SystemPrompt)
		}

		// Clear conversation if requested
		if params.ClearConversation {
			conv.Clear()
		}

		messages = conv.GetMessages()
	}

	// Ensure we have messages
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages provided")
	}

	// Build request
	req := ChatRequest{
		Model:    c.model,
		Messages: messages,
	}

	// Set optional parameters
	if params.Temperature > 0 {
		req.Temperature = params.Temperature
	} else {
		req.Temperature = 0.7 // Default temperature
	}

	if params.MaxTokens > 0 {
		req.MaxTokens = params.MaxTokens
	} else {
		req.MaxTokens = 2048 // Default max tokens
	}

	if params.TopP > 0 {
		req.TopP = params.TopP
	}

	// Send request
	data, err := c.doRequest(ctx, "/text/chatcompletion_v2", req)
	if err != nil {
		return nil, err
	}

	// Parse response
	var response ChatResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Add assistant response to conversation history
	if params.Messages == nil {
		if len(response.Choices) > 0 {
			conv := c.getOrCreateConversation(params.UserID)
			conv.AddMessage("assistant", response.Choices[0].Message.Content)
		}
	}

	return &response, nil
}

// StreamChat sends a chat completion request with streaming response.
func (c *Client) StreamChat(ctx context.Context, params ChatParams, onChunk func(string) error) error {
	// Build messages
	messages := params.Messages

	if messages == nil {
		conv := c.getOrCreateConversation(params.UserID)
		messages = conv.GetMessages()
	}

	if len(messages) == 0 {
		return fmt.Errorf("no messages provided")
	}

	req := ChatRequest{
		Model:    c.model,
		Messages: messages,
		Stream:   true,
	}

	if params.Temperature > 0 {
		req.Temperature = params.Temperature
	} else {
		req.Temperature = 0.7
	}

	if params.MaxTokens > 0 {
		req.MaxTokens = params.MaxTokens
	}

	if params.TopP > 0 {
		req.TopP = params.TopP
	}

	// Create streaming request
	jsonData, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/text/chatcompletion_v2", bytes.NewReader(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		var errResp ErrorResponse
		if json.Unmarshal(data, &errResp) == nil {
			return fmt.Errorf("minimax API error: %s", errResp.Error.Message)
		}
		return fmt.Errorf("minimax API error: %s", string(data))
	}

	// Read streaming response
	decoder := json.NewDecoder(resp.Body)
	for {
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("failed to decode chunk: %w", err)
		}

		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				if err := onChunk(content); err != nil {
					return err
				}

				// Add to conversation history
				if params.Messages == nil {
					conv := c.getOrCreateConversation(params.UserID)
					conv.AddMessage("assistant", content)
				}
			}

			if chunk.Choices[0].FinishReason != "" {
				break
			}
		}
	}

	return nil
}

// ClearConversation clears the conversation history for a user.
func (c *Client) ClearConversation(userID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.conversations, userID)
}

// GetConversation returns the conversation history for a user.
func (c *Client) GetConversation(userID int64) []Message {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if conv, ok := c.conversations[userID]; ok {
		return conv.GetMessages()
	}
	return nil
}

func (c *Client) getOrCreateConversation(userID int64) *Conversation {
	c.mu.Lock()
	defer c.mu.Unlock()

	if conv, ok := c.conversations[userID]; ok {
		return conv
	}

	conv := &Conversation{}
	c.conversations[userID] = conv
	return conv
}

func (c *Client) doRequest(ctx context.Context, path string, body interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+path, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)

	if c.debug {
		c.logger.Debug("Minimax Request: %s %s", httpReq.Method, httpReq.URL.String())
		c.logger.Debug("Minimax Body: %s", string(jsonData))
	}

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.httpClient.Do(httpReq.WithContext(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if c.debug {
		c.logger.Debug("Minimax Response: %s", string(data))
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if json.Unmarshal(data, &errResp) == nil {
			return nil, fmt.Errorf("minimax API error (%d): %s - %s", resp.StatusCode, errResp.Error.Type, errResp.Error.Message)
		}
		return nil, fmt.Errorf("minimax API error (%d): %s", resp.StatusCode, string(data))
	}

	return data, nil
}
