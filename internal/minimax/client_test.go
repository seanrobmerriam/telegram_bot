package minimax

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		apiKey  string
		wantErr bool
	}{
		{
			name:    "valid API key",
			apiKey:  "test_api_key_123",
			wantErr: false,
		},
		{
			name:    "empty API key",
			apiKey:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.apiKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && client == nil {
				t.Error("NewClient() returned nil client")
			}
		})
	}
}

func TestClientOptions(t *testing.T) {
	client, err := NewClient(
		"test_api_key",
		WithBaseURL("https://custom.api.minimax.chat/v1"),
		WithModel("custom-model"),
		WithTimeout(30*time.Second),
	)

	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	if client.baseURL != "https://custom.api.minimax.chat/v1" {
		t.Errorf("baseURL = %s, expect custom URL", client.baseURL)
	}

	if client.model != "custom-model" {
		t.Errorf("model = %s, expect 'custom-model'", client.model)
	}

	if client.timeout != 30*time.Second {
		t.Errorf("timeout = %v, expect 30s", client.timeout)
	}
}

func TestConversation(t *testing.T) {
	conv := &Conversation{}

	// Add some messages
	conv.AddMessage("user", "Hello")
	conv.AddMessage("assistant", "Hi there!")

	messages := conv.GetMessages()
	if len(messages) != 2 {
		t.Errorf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0].Role != "user" {
		t.Errorf("First message role = %s, expect 'user'", messages[0].Role)
	}

	if messages[0].Content != "Hello" {
		t.Errorf("First message content = %s, expect 'Hello'", messages[0].Content)
	}

	if messages[1].Role != "assistant" {
		t.Errorf("Second message role = %s, expect 'assistant'", messages[1].Role)
	}

	// Test clear
	conv.Clear()
	messages = conv.GetMessages()
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", len(messages))
	}

	// Test system prompt
	conv.SetSystem("You are a helpful assistant.")
	if conv.System != "You are a helpful assistant." {
		t.Errorf("System = %s, expect system prompt", conv.System)
	}
}

func TestChatParams(t *testing.T) {
	params := ChatParams{
		UserID:            12345,
		Messages:          []Message{{Role: "user", Content: "Test"}},
		Temperature:       0.8,
		MaxTokens:         1024,
		TopP:              0.9,
		ClearConversation: true,
		SystemPrompt:      "Custom system prompt",
	}

	if params.UserID != 12345 {
		t.Errorf("UserID = %d, expect 12345", params.UserID)
	}

	if len(params.Messages) != 1 {
		t.Errorf("Messages length = %d, expect 1", len(params.Messages))
	}

	if params.Temperature != 0.8 {
		t.Errorf("Temperature = %f, expect 0.8", params.Temperature)
	}

	if params.MaxTokens != 1024 {
		t.Errorf("MaxTokens = %d, expect 1024", params.MaxTokens)
	}

	if params.TopP != 0.9 {
		t.Errorf("TopP = %f, expect 0.9", params.TopP)
	}

	if !params.ClearConversation {
		t.Error("ClearConversation should be true")
	}

	if params.SystemPrompt != "Custom system prompt" {
		t.Errorf("SystemPrompt = %s, expect 'Custom system prompt'", params.SystemPrompt)
	}
}

func TestChatRequest(t *testing.T) {
	req := ChatRequest{
		Model: "abab5.5-chat",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
		},
		Temperature: 0.7,
		MaxTokens:   2048,
		TopP:        1.0,
		Stream:      false,
	}

	if req.Model != "abab5.5-chat" {
		t.Errorf("Model = %s, expect 'abab5.5-chat'", req.Model)
	}

	if len(req.Messages) != 2 {
		t.Errorf("Messages length = %d, expect 2", len(req.Messages))
	}

	if req.Temperature != 0.7 {
		t.Errorf("Temperature = %f, expect 0.7", req.Temperature)
	}

	if req.MaxTokens != 2048 {
		t.Errorf("MaxTokens = %d, expect 2048", req.MaxTokens)
	}

	if req.Stream != false {
		t.Error("Stream should be false")
	}
}

func TestChatResponse(t *testing.T) {
	resp := &ChatResponse{
		ID:      "chatcmpl-123",
		Object:  "chat.completion",
		Created: 1234567890,
		Model:   "abab5.5-chat",
		Choices: []Choice{
			{
				Index:        0,
				Message:      Message{Role: "assistant", Content: "Hello!"},
				FinishReason: "stop",
			},
		},
		Usage: Usage{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:      15,
		},
	}

	if resp.ID != "chatcmpl-123" {
		t.Errorf("ID = %s, expect 'chatcmpl-123'", resp.ID)
	}

	if len(resp.Choices) != 1 {
		t.Errorf("Choices length = %d, expect 1", len(resp.Choices))
	}

	if resp.Choices[0].Message.Content != "Hello!" {
		t.Errorf("Message content = %s, expect 'Hello!'", resp.Choices[0].Message.Content)
	}

	if resp.Choices[0].FinishReason != "stop" {
		t.Errorf("FinishReason = %s, expect 'stop'", resp.Choices[0].FinishReason)
	}

	if resp.Usage.TotalTokens != 15 {
		t.Errorf("TotalTokens = %d, expect 15", resp.Usage.TotalTokens)
	}
}

func TestErrorResponse(t *testing.T) {
	errResp := &ErrorResponse{
		Error: ErrorDetail{
			Message: "Invalid API key",
			Type:    "invalid_request_error",
			Code:    "401",
		},
	}

	if errResp.Error.Message != "Invalid API key" {
		t.Errorf("Error message = %s, expect 'Invalid API key'", errResp.Error.Message)
	}

	if errResp.Error.Type != "invalid_request_error" {
		t.Errorf("Error type = %s, expect 'invalid_request_error'", errResp.Error.Type)
	}

	if errResp.Error.Code != "401" {
		t.Errorf("Error code = %s, expect '401'", errResp.Error.Code)
	}
}

func TestGetOrCreateConversation(t *testing.T) {
	client := &Client{
		conversations: make(map[int64]*Conversation),
	}

	// Get non-existent conversation - should create new one
	conv1 := client.getOrCreateConversation(123)
	if conv1 == nil {
		t.Error("Conversation should not be nil")
	}

	// Get existing conversation - should return same one
	conv2 := client.getOrCreateConversation(123)
	if conv1 != conv2 {
		t.Error("Should return same conversation for same user")
	}

	// Different user should get different conversation
	conv3 := client.getOrCreateConversation(456)
	if conv1 == conv3 {
		t.Error("Different users should get different conversations")
	}
}

func TestClearConversation(t *testing.T) {
	client := &Client{
		conversations: make(map[int64]*Conversation),
	}

	// Add conversation for user
	conv := client.getOrCreateConversation(123)
	conv.AddMessage("user", "Hello")

	// Clear conversation
	client.ClearConversation(123)

	// Verify it's cleared
	messages := client.GetConversation(123)
	if len(messages) != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", len(messages))
	}

	// Clearing non-existent conversation should not panic
	client.ClearConversation(999)
}

func TestGetConversation(t *testing.T) {
	client := &Client{
		conversations: make(map[int64]*Conversation),
	}

	// Get non-existent conversation
	messages := client.GetConversation(123)
	if messages != nil {
		t.Error("Should return nil for non-existent conversation")
	}

	// Add conversation
	conv := client.getOrCreateConversation(123)
	conv.AddMessage("user", "Test")

	// Get existing conversation
	messages = client.GetConversation(123)
	if len(messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(messages))
	}
}
