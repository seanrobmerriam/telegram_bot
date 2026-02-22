// Package telegram provides a client for interacting with the Telegram Bot API.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/minimax-agent/telegram-bot/pkg/logger"
)

// APIURL is the base URL for the Telegram Bot API.
const APIURL = "https://api.telegram.org"

// Client represents a Telegram Bot API client.
type Client struct {
	mu         sync.RWMutex
	token      string
	baseURL    string
	httpClient *http.Client
	debug      bool
	logger     *logger.Logger

	// Bot information (cached after calling GetMe)
	botInfo *User

	// Update handling
	updateCh  chan Update
	offset    int64
	connected bool
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

// NewClient creates a new Telegram API client.
func NewClient(token string, opts ...ClientOption) (*Client, error) {
	if token == "" {
		return nil, errors.New("bot token is required")
	}

	client := &Client{
		token:      token,
		baseURL:    APIURL + "/bot" + token,
		httpClient: &http.Client{Timeout: 60 * time.Second},
		logger:     logger.Default(),
		updateCh:   make(chan Update, 100),
		stopChan:   make(chan struct{}),
	}

	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// ClientOption configures the Telegram client.
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithBaseURL sets a custom base URL for the Telegram API.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL + "/bot" + c.token
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

// WithUpdateChannel sets a custom channel for receiving updates.
func WithUpdateChannel(ch chan Update) ClientOption {
	return func(c *Client) {
		c.updateCh = ch
	}
}

// APIError represents a Telegram API error.
type APIError struct {
	Code        int    `json:"error_code"`
	Description string `json:"description"`
}

// Error returns a string representation of the API error.
func (e *APIError) Error() string {
	return fmt.Sprintf("telegram API error %d: %s", e.Code, e.Description)
}

// Response represents a generic Telegram API response.
type Response struct {
	OK          bool                `json:"ok"`
	ErrorCode   int                 `json:"error_code"`
	Description string              `json:"description"`
	Result      json.RawMessage     `json:"result"`
	Parameters  *ResponseParameters `json:"parameters"`
}

// ResponseParameters contains information about why a request was unsuccessful.
type ResponseParameters struct {
	MigrateToChatID int64 `json:"migrate_to_chat_id"` // Optional. Retry after this number of seconds
	RetryAfter      int   `json:"retry_after"`        // Optional. The group has been migrated to a supergroup
}

// User represents a Telegram user or bot.
type User struct {
	ID                      int64  `json:"id"`
	IsBot                   bool   `json:"is_bot"`
	FirstName               string `json:"first_name"`
	LastName                string `json:"last_name"`
	Username                string `json:"username"`
	LanguageCode            string `json:"language_code"`
	IsPremium               bool   `json:"is_premium"`
	AddedToAttachmentMenu   bool   `json:"added_to_attachment_menu"`
	CanJoinGroups           bool   `json:"can_join_groups"`
	CanReadAllGroupMessages bool   `json:"can_read_all_group_messages"`
	SupportsInlineQueries   bool   `json:"supports_inline_queries"`
}

// Chat represents a Telegram chat.
type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// Message represents a Telegram message.
type Message struct {
	MessageID       int64           `json:"message_id"`
	MessageThreadID int64           `json:"message_thread_id"`
	From            *User           `json:"from"`
	Date            int             `json:"date"`
	Chat            *Chat           `json:"chat"`
	Text            string          `json:"text"`
	Entities        []MessageEntity `json:"entities"`
	Caption         string          `json:"caption"`
	CaptionEntities []MessageEntity `json:"caption_entities"`
}

// MessageEntity represents a special entity in a text message.
type MessageEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
	URL    string `json:"url"`
	User   *User  `json:"user"`
}

// Update represents an incoming update from Telegram.
type Update struct {
	UpdateID           int64               `json:"update_id"`
	Message            *Message            `json:"message"`
	EditedMessage      *Message            `json:"edited_message"`
	InlineQuery        *InlineQuery        `json:"inline_query"`
	CallbackQuery      *CallbackQuery      `json:"callback_query"`
	ChannelPost        *Message            `json:"channel_post"`
	EditedChannelPost  *Message            `json:"edited_channel_post"`
	ChosenInlineResult *ChosenInlineResult `json:"chosen_inline_result"`
	ShippingQuery      *ShippingQuery      `json:"shipping_query"`
	PreCheckoutQuery   *PreCheckoutQuery   `json:"pre_checkout_query"`
	Poll               *Poll               `json:"poll"`
	PollAnswer         *PollAnswer         `json:"poll_answer"`
	MyChatMember       *ChatMemberUpdated  `json:"my_chat_member"`
	ChatMember         *ChatMemberUpdated  `json:"chat_member"`
	ChatJoinRequest    *ChatJoinRequest    `json:"chat_join_request"`
}

// InlineQuery represents an incoming inline query.
type InlineQuery struct {
	ID       string    `json:"id"`
	From     *User     `json:"from"`
	Query    string    `json:"query"`
	Offset   string    `json:"offset"`
	ChatType string    `json:"chat_type"`
	Location *Location `json:"location"`
}

// Location represents a geographical location.
type Location struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}

// ChosenInlineResult represents a result of an inline query chosen by the user.
type ChosenInlineResult struct {
	ResultID        string    `json:"result_id"`
	From            *User     `json:"from"`
	Location        *Location `json:"location"`
	InlineMessageID string    `json:"inline_message_id"`
	Query           string    `json:"query"`
}

// CallbackQuery represents an incoming callback query.
type CallbackQuery struct {
	ID              string   `json:"id"`
	From            *User    `json:"from"`
	ChatInstance    string   `json:"chat_instance"`
	Data            string   `json:"data"`
	GameShortName   string   `json:"game_short_name"`
	InlineMessageID string   `json:"inline_message_id"`
	Message         *Message `json:"message"`
}

// ShippingQuery represents an incoming shipping query.
type ShippingQuery struct {
	ID              string           `json:"id"`
	From            *User            `json:"from"`
	InvoicePayload  string           `json:"invoice_payload"`
	ShippingAddress *ShippingAddress `json:"shipping_address"`
}

// ShippingAddress represents a shipping address.
type ShippingAddress struct {
	CountryCode string `json:"country_code"`
	State       string `json:"state"`
	City        string `json:"city"`
	StreetLine1 string `json:"street_line1"`
	StreetLine2 string `json:"street_line2"`
	PostCode    string `json:"post_code"`
}

// PreCheckoutQuery represents an incoming pre-checkout query.
type PreCheckoutQuery struct {
	ID               string     `json:"id"`
	From             *User      `json:"from"`
	Currency         string     `json:"currency"`
	TotalAmount      int        `json:"total_amount"`
	InvoicePayload   string     `json:"invoice_payload"`
	ShippingOptionID string     `json:"shipping_option_id"`
	OrderInfo        *OrderInfo `json:"order_info"`
}

// OrderInfo represents order information.
type OrderInfo struct {
	Name            string           `json:"name"`
	PhoneNumber     string           `json:"phone_number"`
	Email           string           `json:"email"`
	ShippingAddress *ShippingAddress `json:"shipping_address"`
}

// Poll represents a poll.
type Poll struct {
	ID                    string       `json:"id"`
	Question              string       `json:"question"`
	TotalVoterCount       int          `json:"total_voter_count"`
	IsClosed              bool         `json:"is_closed"`
	IsAnonymous           bool         `json:"is_anonymous"`
	Type                  string       `json:"type"`
	AllowsMultipleAnswers bool         `json:"allows_multiple_answers"`
	CorrectOptionID       int          `json:"correct_option_id"`
	Explanation           string       `json:"explanation"`
	Options               []PollOption `json:"options"`
}

// PollOption represents an answer option in a poll.
type PollOption struct {
	Text       string `json:"text"`
	VoterCount int    `json:"voter_count"`
}

// PollAnswer represents an answer in a poll.
type PollAnswer struct {
	PollID    string `json:"poll_id"`
	User      *User  `json:"user"`
	OptionIDs []int  `json:"option_ids"`
}

// ChatMemberUpdated represents changes in a chat member.
type ChatMemberUpdated struct {
	Chat          *Chat           `json:"chat"`
	From          *User           `json:"from"`
	Date          int             `json:"date"`
	OldChatMember *ChatMember     `json:"old_chat_member"`
	NewChatMember *ChatMember     `json:"new_chat_member"`
	InviteLink    *ChatInviteLink `json:"invite_link"`
}

// ChatInviteLink represents an invite link for a chat.
type ChatInviteLink struct {
	InviteLink         string `json:"invite_link"`
	Creator            *User  `json:"creator"`
	IsPrimary          bool   `json:"is_primary"`
	IsRevoked          bool   `json:"is_revoked"`
	ExpireDate         int    `json:"expire_date"`
	MemberLimit        int    `json:"member_limit"`
	Name               string `json:"name"`
	CreatesJoinRequest bool   `json:"creates_join_request"`
}

// ChatJoinRequest represents a join request to a chat.
type ChatJoinRequest struct {
	Chat       *Chat           `json:"chat"`
	From       *User           `json:"from"`
	Date       int             `json:"date"`
	Bio        string          `json:"bio"`
	InviteLink *ChatInviteLink `json:"invite_link"`
}

// ChatMember represents information about a chat member.
type ChatMember struct {
	Status                string `json:"status"`
	User                  *User  `json:"user"`
	Title                 string `json:"title"`
	UntilDate             int    `json:"until_date"`
	CanBeEdited           bool   `json:"can_be_edited"`
	CanManageChat         bool   `json:"can_manage_chat"`
	CanChangeInfo         bool   `json:"can_change_info"`
	CanDeleteMessages     bool   `json:"can_delete_messages"`
	CanInviteUsers        bool   `json:"can_invite_users"`
	CanRestrictMembers    bool   `json:"can_restrict_members"`
	CanPinMessages        bool   `json:"can_pin_messages"`
	CanPromoteMembers     bool   `json:"can_promote_members"`
	CanSendMessages       bool   `json:"can_send_messages"`
	CanSendMediaMessages  bool   `json:"can_send_media_messages"`
	CanSendOtherMessages  bool   `json:"can_send_other_messages"`
	CanAddWebPagePreviews bool   `json:"can_add_web_page_previews"`
}

// SendMessageParams contains parameters for sending a message.
type SendMessageParams struct {
	ChatID                   interface{}     `json:"chat_id"`
	Text                     string          `json:"text"`
	ParseMode                string          `json:"parse_mode,omitempty"`
	Entities                 []MessageEntity `json:"entities,omitempty"`
	DisableWebPagePreview    bool            `json:"disable_web_page_preview,omitempty"`
	DisableNotification      bool            `json:"disable_notification,omitempty"`
	ReplyToMessageID         int64           `json:"reply_to_message_id,omitempty"`
	AllowSendingWithoutReply bool            `json:"allow_sending_without_reply,omitempty"`
	ReplyMarkup              interface{}     `json:"reply_markup,omitempty"`
}

// SendMessage sends a message to a chat.
func (c *Client) SendMessage(ctx context.Context, params SendMessageParams) (*Message, error) {
	data, err := c.doRequest("sendMessage", params)
	if err != nil {
		return nil, err
	}

	var result Message
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// AnswerCallbackQueryParams contains parameters for answering a callback query.
type AnswerCallbackQueryParams struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
	ShowAlert       bool   `json:"show_alert,omitempty"`
	URL             string `json:"url,omitempty"`
	CacheTime       int    `json:"cache_time,omitempty"`
}

// AnswerCallbackQuery answers a callback query.
func (c *Client) AnswerCallbackQuery(ctx context.Context, params AnswerCallbackQueryParams) (bool, error) {
	data, err := c.doRequest("answerCallbackQuery", params)
	if err != nil {
		return false, err
	}

	var result bool
	if err := json.Unmarshal(data, &result); err != nil {
		return false, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// DeleteMessageParams contains parameters for deleting a message.
type DeleteMessageParams struct {
	ChatID    interface{} `json:"chat_id"`
	MessageID int64       `json:"message_id"`
}

// DeleteMessage deletes a message from a chat.
func (c *Client) DeleteMessage(ctx context.Context, chatID interface{}, messageID int64) (bool, error) {
	params := DeleteMessageParams{
		ChatID:    chatID,
		MessageID: messageID,
	}

	data, err := c.doRequest("deleteMessage", params)
	if err != nil {
		return false, err
	}

	var result bool
	if err := json.Unmarshal(data, &result); err != nil {
		return false, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// GetChatMemberParams contains parameters for getting a chat member.
type GetChatMemberParams struct {
	ChatID interface{} `json:"chat_id"`
	UserID int64       `json:"user_id"`
}

// GetChatMember gets information about a member of a chat.
func (c *Client) GetChatMember(ctx context.Context, params GetChatMemberParams) (*ChatMember, error) {
	data, err := c.doRequest("getChatMember", params)
	if err != nil {
		return nil, err
	}

	var result ChatMember
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &result, nil
}

// GetMe gets information about the bot.
func (c *Client) GetMe(ctx context.Context) (*User, error) {
	// Use cached bot info if available
	c.mu.RLock()
	if c.botInfo != nil {
		c.mu.RUnlock()
		return c.botInfo, nil
	}
	c.mu.RUnlock()

	data, err := c.doRequest("getMe", nil)
	if err != nil {
		return nil, err
	}

	var botInfo User
	if err := json.Unmarshal(data, &botInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	c.mu.Lock()
	c.botInfo = &botInfo
	c.mu.Unlock()

	return &botInfo, nil
}

// GetUpdatesParams contains parameters for getting updates.
type GetUpdatesParams struct {
	Offset         int64    `json:"offset,omitempty"`
	Limit          int      `json:"limit,omitempty"`
	Timeout        int      `json:"timeout,omitempty"`
	AllowedUpdates []string `json:"allowed_updates,omitempty"`
}

// GetUpdates gets updates from Telegram.
func (c *Client) GetUpdates(ctx context.Context, params GetUpdatesParams) ([]Update, error) {
	data, err := c.doRequest("getUpdates", params)
	if err != nil {
		return nil, err
	}

	var result []Update
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// SetWebhookParams contains parameters for setting a webhook.
type SetWebhookParams struct {
	URL                string      `json:"url"`
	Certificate        interface{} `json:"certificate,omitempty"`
	IPAddress          string      `json:"ip_address,omitempty"`
	MaxConnections     int         `json:"max_connections,omitempty"`
	AllowedUpdates     []string    `json:"allowed_updates,omitempty"`
	DropPendingUpdates bool        `json:"drop_pending_updates,omitempty"`
	SecretToken        string      `json:"secret_token,omitempty"`
}

// SetWebhook sets a webhook for the bot.
func (c *Client) SetWebhook(ctx context.Context, params SetWebhookParams) (bool, error) {
	data, err := c.doRequest("setWebhook", params)
	if err != nil {
		return false, err
	}

	var result bool
	if err := json.Unmarshal(data, &result); err != nil {
		return false, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// DeleteWebhook deletes the webhook.
func (c *Client) DeleteWebhook(ctx context.Context, dropPendingUpdates bool) (bool, error) {
	params := map[string]interface{}{
		"drop_pending_updates": dropPendingUpdates,
	}
	data, err := c.doRequest("deleteWebhook", params)
	if err != nil {
		return false, err
	}

	var result bool
	if err := json.Unmarshal(data, &result); err != nil {
		return false, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}

// StartLongPolling starts long polling for updates.
func (c *Client) StartLongPolling(ctx context.Context, opts ...GetUpdatesOption) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return errors.New("long polling already started")
	}
	c.connected = true
	c.mu.Unlock()

	c.wg.Add(1)
	go c.longPollingLoop(ctx, opts...)

	return nil
}

// StopLongPolling stops long polling.
func (c *Client) StopLongPolling() error {
	c.mu.Lock()
	if !c.connected {
		c.mu.Unlock()
		return errors.New("long polling not started")
	}
	c.mu.Unlock()

	close(c.stopChan)
	c.wg.Wait()

	c.mu.Lock()
	c.connected = false
	c.mu.Unlock()

	return nil
}

// GetUpdateChannel returns a channel for receiving updates.
func (c *Client) GetUpdateChannel() <-chan Update {
	return c.updateCh
}

// GetUpdatesOption configures getUpdates parameters.
type GetUpdatesOption func(*GetUpdatesParams)

// WithOffset sets the offset for getUpdates.
func WithOffset(offset int64) GetUpdatesOption {
	return func(p *GetUpdatesParams) {
		p.Offset = offset
	}
}

// WithLimit sets the limit for getUpdates.
func WithLimit(limit int) GetUpdatesOption {
	return func(p *GetUpdatesParams) {
		p.Limit = limit
	}
}

// WithTimeout sets the timeout for getUpdates.
func WithTimeout(timeout int) GetUpdatesOption {
	return func(p *GetUpdatesParams) {
		p.Timeout = timeout
	}
}

// WithAllowedUpdates sets the allowed updates for getUpdates.
func WithAllowedUpdates(updates []string) GetUpdatesOption {
	return func(p *GetUpdatesParams) {
		p.AllowedUpdates = updates
	}
}

func (c *Client) longPollingLoop(ctx context.Context, opts ...GetUpdatesOption) {
	defer c.wg.Done()

	params := &GetUpdatesParams{
		Timeout: 30,
		Limit:   100,
	}

	for _, opt := range opts {
		opt(params)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopChan:
			return
		default:
		}

		// Set offset to the next update after the last processed
		params.Offset = c.offset

		updates, err := c.GetUpdates(ctx, *params)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-c.stopChan:
				return
			default:
			}

			c.logger.Error("Error getting updates: %v", err)
			time.Sleep(time.Second)
			continue
		}

		for _, update := range updates {
			c.offset = update.UpdateID + 1

			select {
			case c.updateCh <- update:
			case <-ctx.Done():
				return
			case <-c.stopChan:
				return
			}
		}
	}
}

// doRequest performs a request to the Telegram API.
func (c *Client) doRequest(method string, params interface{}) (json.RawMessage, error) {
	var body io.Reader

	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/"+method, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if c.debug {
		c.logger.Debug("Request: %s %s", req.Method, req.URL.String())
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if c.debug {
		c.logger.Debug("Response: %s", string(data))
	}

	var result Response
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if !result.OK {
		return nil, &APIError{
			Code:        result.ErrorCode,
			Description: result.Description,
		}
	}

	return result.Result, nil
}

// FormatChatID converts a chat ID to the appropriate format for the API.
func FormatChatID(chatID interface{}) string {
	switch v := chatID.(type) {
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.Itoa(v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", chatID)
	}
}

// IsValidChatID checks if the chat ID is valid.
func IsValidChatID(chatID interface{}) bool {
	switch v := chatID.(type) {
	case int64:
		return v != 0
	case int:
		return v != 0
	case string:
		return strings.TrimSpace(v) != ""
	default:
		return false
	}
}
