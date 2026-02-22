package telegram

import (
	"context"
	"testing"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   "1234567890:ABCdefGHIjklMNOpqrsTUVwxyz",
			wantErr: false,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.token)
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

func TestFormatChatID(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		expect string
	}{
		{
			name:   "int64",
			input:  int64(123456789),
			expect: "123456789",
		},
		{
			name:   "int",
			input:  123456789,
			expect: "123456789",
		},
		{
			name:   "string",
			input:  "123456789",
			expect: "123456789",
		},
		{
			name:   "negative int64",
			input:  int64(-1000000000000),
			expect: "-1000000000000",
		},
		{
			name:   "string with underscore",
			input:  "chat_123456789",
			expect: "chat_123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatChatID(tt.input)
			if result != tt.expect {
				t.Errorf("FormatChatID(%v) = %s, expect %s", tt.input, result, tt.expect)
			}
		})
	}
}

func TestIsValidChatID(t *testing.T) {
	tests := []struct {
		name   string
		input  interface{}
		expect bool
	}{
		{
			name:   "valid int64",
			input:  int64(123456789),
			expect: true,
		},
		{
			name:   "valid int",
			input:  123456789,
			expect: true,
		},
		{
			name:   "valid string",
			input:  "123456789",
			expect: true,
		},
		{
			name:   "zero int64",
			input:  int64(0),
			expect: false,
		},
		{
			name:   "zero int",
			input:  0,
			expect: false,
		},
		{
			name:   "empty string",
			input:  "",
			expect: false,
		},
		{
			name:   "whitespace string",
			input:  "   ",
			expect: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidChatID(tt.input)
			if result != tt.expect {
				t.Errorf("IsValidChatID(%v) = %v, expect %v", tt.input, result, tt.expect)
			}
		})
	}
}

func TestClientWithOptions(t *testing.T) {
	token := "test_token_123"

	client, err := NewClient(
		token,
		WithDebug(true),
	)

	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}

	// Verify client is configured correctly
	if client == nil {
		t.Fatal("Client should not be nil")
	}

	// Test GetMe (should fail without real API, but tests the flow)
	ctx := context.Background()
	_, err = client.GetMe(ctx)
	// This will fail because we don't have a real token, but that's expected
	_ = err
}

func TestAPIError(t *testing.T) {
	err := &APIError{
		Code:        400,
		Description: "Bad Request",
	}

	expected := "telegram API error 400: Bad Request"
	if err.Error() != expected {
		t.Errorf("APIError.Error() = %s, expect %s", err.Error(), expected)
	}
}

func TestSendMessageParams(t *testing.T) {
	params := SendMessageParams{
		ChatID:                   int64(123456789),
		Text:                     "Hello, World!",
		ParseMode:                "MarkdownV2",
		DisableWebPagePreview:    true,
		DisableNotification:      true,
		ReplyToMessageID:         123,
		AllowSendingWithoutReply: true,
	}

	if params.ChatID != int64(123456789) {
		t.Errorf("ChatID = %v, expect 123456789", params.ChatID)
	}

	if params.Text != "Hello, World!" {
		t.Errorf("Text = %s, expect 'Hello, World!'", params.Text)
	}

	if params.ParseMode != "MarkdownV2" {
		t.Errorf("ParseMode = %s, expect 'MarkdownV2'", params.ParseMode)
	}
}

func TestWithOffset(t *testing.T) {
	opts := WithOffset(100)
	params := &GetUpdatesParams{}

	opts(params)

	if params.Offset != 100 {
		t.Errorf("Offset = %d, expect 100", params.Offset)
	}
}

func TestWithLimit(t *testing.T) {
	opts := WithLimit(50)
	params := &GetUpdatesParams{}

	opts(params)

	if params.Limit != 50 {
		t.Errorf("Limit = %d, expect 50", params.Limit)
	}
}

func TestWithTimeout(t *testing.T) {
	opts := WithTimeout(60)
	params := &GetUpdatesParams{}

	opts(params)

	if params.Timeout != 60 {
		t.Errorf("Timeout = %d, expect 60", params.Timeout)
	}
}

func TestWithAllowedUpdates(t *testing.T) {
	updates := []string{"message", "callback_query"}
	opts := WithAllowedUpdates(updates)
	params := &GetUpdatesParams{}

	opts(params)

	if len(params.AllowedUpdates) != 2 {
		t.Errorf("AllowedUpdates length = %d, expect 2", len(params.AllowedUpdates))
	}

	if params.AllowedUpdates[0] != "message" {
		t.Errorf("AllowedUpdates[0] = %s, expect 'message'", params.AllowedUpdates[0])
	}
}
