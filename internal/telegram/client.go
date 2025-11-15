package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/zelenin/go-tdlib/client"
	"tdlib-go/internal/config"
	"tdlib-go/pkg/models"
)

// Client wraps the TDLib client with additional functionality
type Client struct {
	tdClient  *client.Client
	config    *config.Config
	logger    *logrus.Logger
	handlers  []MessageHandler
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
	connected bool
}

// MessageHandler is a function that processes incoming messages
type MessageHandler func(msg *models.Message) error

// NewClient creates a new Telegram client
func NewClient(cfg *config.Config, logger *logrus.Logger) (*Client, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Prepare TDLib parameters for QR authorizer
	tdlibCfg := cfg.TDLib
	tdlibParams := &client.SetTdlibParametersRequest{
		UseTestDc:           cfg.Telegram.UseTestDC,
		DatabaseDirectory:   tdlibCfg.DatabaseDirectory,
		FilesDirectory:      tdlibCfg.FilesDirectory,
		UseFileDatabase:     tdlibCfg.UseFileDatabase,
		UseChatInfoDatabase: tdlibCfg.UseChatInfoDB,
		UseMessageDatabase:  tdlibCfg.UseMessageDB,
		UseSecretChats:      false,
		ApiId:               cfg.Telegram.APIID,
		ApiHash:             cfg.Telegram.APIHash,
		SystemLanguageCode:  tdlibCfg.SystemLanguage,
		DeviceModel:         tdlibCfg.DeviceModel,
		SystemVersion:       tdlibCfg.SystemVersion,
		ApplicationVersion:  tdlibCfg.AppVersion,
	}

	// Use QR code authorizer
	authorizer := client.QrAuthorizer(tdlibParams, func(link string) error {
		logger.Infof("\n========================================")
		logger.Infof("QR Code Authentication")
		logger.Infof("========================================")
		logger.Infof("Please scan this QR code with your Telegram app:")
		logger.Infof("\nLink: %s", link)
		logger.Infof("\nOr open this link in your browser:")
		logger.Infof("%s", link)
		logger.Infof("========================================\n")
		return nil
	})

	// Create TDLib client
	tdClient, err := client.NewClient(authorizer, client.WithLogVerbosity(&client.SetLogVerbosityLevelRequest{
		NewVerbosityLevel: 1,
	}))

	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create TDLib client: %w", err)
	}

	c := &Client{
		tdClient:  tdClient,
		config:    cfg,
		logger:    logger,
		handlers:  make([]MessageHandler, 0),
		ctx:       ctx,
		cancel:    cancel,
		connected: false,
	}

	return c, nil
}

// Start initializes and authenticates the client
func (c *Client) Start() error {
	c.logger.Info("Starting Telegram client...")

	// TDLib parameters are already set by QrAuthorizer
	// Wait for authentication to complete
	if err := c.waitForAuthentication(); err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	c.connected = true
	c.logger.Info("Telegram client started successfully")

	return nil
}

// setTDLibParameters sets TDLib configuration parameters
func (c *Client) setTDLibParameters() error {
	tdlibCfg := c.config.TDLib

	params := &client.SetTdlibParametersRequest{
		UseTestDc:           c.config.Telegram.UseTestDC,
		DatabaseDirectory:   tdlibCfg.DatabaseDirectory,
		FilesDirectory:      tdlibCfg.FilesDirectory,
		UseFileDatabase:     tdlibCfg.UseFileDatabase,
		UseChatInfoDatabase: tdlibCfg.UseChatInfoDB,
		UseMessageDatabase:  tdlibCfg.UseMessageDB,
		UseSecretChats:      false,
		ApiId:               c.config.Telegram.APIID,
		ApiHash:             c.config.Telegram.APIHash,
		SystemLanguageCode:  tdlibCfg.SystemLanguage,
		DeviceModel:         tdlibCfg.DeviceModel,
		SystemVersion:       tdlibCfg.SystemVersion,
		ApplicationVersion:  tdlibCfg.AppVersion,
	}

	_, err := c.tdClient.SetTdlibParameters(params)
	return err
}

// waitForAuthentication waits for the authentication flow to complete
func (c *Client) waitForAuthentication() error {
	// Authentication is handled by the QrAuthorizer
	// User needs to scan the QR code with their Telegram app
	maxRetries := 60 // 2 minutes total for QR code scanning
	for i := 0; i < maxRetries; i++ {
		state, err := c.tdClient.GetAuthorizationState()
		if err != nil {
			return fmt.Errorf("failed to get authorization state: %w", err)
		}

		if _, ok := state.(*client.AuthorizationStateReady); ok {
			c.logger.Info("✓ Authorization successful!")
			return nil
		}

		if i == 0 {
			c.logger.Infof("Current authorization state: %T", state)
		}
		if i%5 == 0 && i > 0 {
			c.logger.Debugf("Waiting for authorization... (%d seconds elapsed)", i*2)
		}
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("authorization timeout after %d seconds", maxRetries*2)
}

// GetChat retrieves chat information by username or ID
func (c *Client) GetChat(identifier string) (*client.Chat, error) {
	// Try to search for the chat
	req := &client.SearchPublicChatRequest{
		Username: identifier,
	}

	chat, err := c.tdClient.SearchPublicChat(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search chat: %w", err)
	}

	return chat, nil
}

// GetChatHistory retrieves message history from a chat
func (c *Client) GetChatHistory(chatID int64, limit int32) ([]*client.Message, error) {
	req := &client.GetChatHistoryRequest{
		ChatId:        chatID,
		FromMessageId: 0,
		Offset:        0,
		Limit:         limit,
		OnlyLocal:     false,
	}

	history, err := c.tdClient.GetChatHistory(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat history: %w", err)
	}

	return history.Messages, nil
}

// JoinChat joins a channel by username, ID, or invite link
func (c *Client) JoinChat(identifier string) (*client.Chat, error) {
	var chat *client.Chat
	var err error

	// Detect the type of identifier
	if isInviteLink(identifier) {
		// Handle invite link (e.g., https://t.me/+wIr66-O-XaxjOWI0 or t.me/joinchat/...)
		chat, err = c.joinByInviteLink(identifier)
		if err != nil {
			return nil, fmt.Errorf("failed to join by invite link: %w", err)
		}
	} else if isChatID(identifier) {
		// Handle channel ID (e.g., -1002233859472)
		chatID, parseErr := parseChatID(identifier)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid chat ID: %w", parseErr)
		}
		chat, err = c.joinByChatID(chatID)
		if err != nil {
			return nil, fmt.Errorf("failed to join by chat ID: %w", err)
		}
	} else {
		// Handle username (existing functionality)
		chat, err = c.GetChat(identifier)
		if err != nil {
			return nil, fmt.Errorf("failed to find chat: %w", err)
		}

		// Join the chat
		joinReq := &client.JoinChatRequest{
			ChatId: chat.Id,
		}

		_, err = c.tdClient.JoinChat(joinReq)
		if err != nil {
			c.logger.Warnf("Join chat returned error (might already be member): %v", err)
		}
	}

	return chat, nil
}

// isInviteLink checks if the identifier is an invite link
func isInviteLink(identifier string) bool {
	return strings.Contains(identifier, "t.me/+") ||
		strings.Contains(identifier, "t.me/joinchat/") ||
		strings.Contains(identifier, "telegram.me/+") ||
		strings.Contains(identifier, "telegram.me/joinchat/")
}

// isChatID checks if the identifier is a numeric chat ID
func isChatID(identifier string) bool {
	_, err := strconv.ParseInt(identifier, 10, 64)
	return err == nil
}

// parseChatID parses a string into a chat ID
func parseChatID(identifier string) (int64, error) {
	return strconv.ParseInt(identifier, 10, 64)
}

// joinByInviteLink joins a chat using an invite link
func (c *Client) joinByInviteLink(link string) (*client.Chat, error) {
	c.logger.Infof("Joining chat by invite link: %s", link)

	// Check the invite link first
	checkReq := &client.CheckChatInviteLinkRequest{
		InviteLink: link,
	}

	linkInfo, err := c.tdClient.CheckChatInviteLink(checkReq)
	if err != nil {
		return nil, fmt.Errorf("failed to check invite link: %w", err)
	}

	c.logger.Infof("Invite link info: %+v", linkInfo)

	// Join by invite link
	joinReq := &client.JoinChatByInviteLinkRequest{
		InviteLink: link,
	}

	chat, err := c.tdClient.JoinChatByInviteLink(joinReq)
	if err != nil {
		return nil, fmt.Errorf("failed to join by invite link: %w", err)
	}

	c.logger.Infof("Successfully joined chat: %s (ID: %d)", chat.Title, chat.Id)

	return chat, nil
}

// joinByChatID joins a chat using its ID
func (c *Client) joinByChatID(chatID int64) (*client.Chat, error) {
	c.logger.Infof("Joining chat by ID: %d", chatID)

	// Get the chat first
	getReq := &client.GetChatRequest{
		ChatId: chatID,
	}

	chat, err := c.tdClient.GetChat(getReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get chat: %w", err)
	}

	// Join the chat
	joinReq := &client.JoinChatRequest{
		ChatId: chatID,
	}

	_, err = c.tdClient.JoinChat(joinReq)
	if err != nil {
		c.logger.Warnf("Join chat returned error (might already be member): %v", err)
	}

	c.logger.Infof("Successfully joined chat: %s (ID: %d)", chat.Title, chat.Id)

	return chat, nil
}

// AddMessageHandler adds a handler for incoming messages
func (c *Client) AddMessageHandler(handler MessageHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers = append(c.handlers, handler)
}

// StartListening starts listening for new messages
func (c *Client) StartListening() error {
	c.logger.Info("Starting message listener...")

	listener := c.tdClient.GetListener()
	defer listener.Close()

	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("Stopping message listener...")
			return nil
		default:
			update := <-listener.Updates
			if update == nil {
				continue
			}

			// Handle different update types
			switch update.GetType() {
			case client.TypeUpdateNewMessage:
				c.handleNewMessage(update.(*client.UpdateNewMessage))
			case client.TypeUpdateMessageContent:
				c.logger.Debug("Message content updated")
			case client.TypeUpdateAuthorizationState:
				authUpdate := update.(*client.UpdateAuthorizationState)
				c.handleAuthorizationStateUpdate(authUpdate)
			}
		}
	}
}

// handleNewMessage processes new message updates
func (c *Client) handleNewMessage(update *client.UpdateNewMessage) {
	msg := update.Message

	// Get chat info to determine if it's a channel
	chat, err := c.tdClient.GetChat(&client.GetChatRequest{ChatId: msg.ChatId})
	if err != nil {
		c.logger.Errorf("Failed to get chat info: %v", err)
		return
	}

	// Process only channel messages (supergroups include channels)
	if _, ok := chat.Type.(*client.ChatTypeSupergroup); !ok {
		return
	}

	// Convert TDLib message to our model
	message := c.convertMessage(msg, chat)

	// Call all registered handlers
	c.mu.RLock()
	handlers := make([]MessageHandler, len(c.handlers))
	copy(handlers, c.handlers)
	c.mu.RUnlock()

	for _, handler := range handlers {
		if err := handler(message); err != nil {
			c.logger.Errorf("Message handler error: %v", err)
		}
	}
}

// handleAuthorizationStateUpdate handles authorization state changes
func (c *Client) handleAuthorizationStateUpdate(update *client.UpdateAuthorizationState) {
	c.logger.Infof("Authorization state changed to: %T", update.AuthorizationState)

	if _, ok := update.AuthorizationState.(*client.AuthorizationStateClosed); ok {
		c.connected = false
		c.logger.Warn("Connection closed by Telegram")
	}
}

// convertMessage converts a TDLib message to our model
func (c *Client) convertMessage(msg *client.Message, chat *client.Chat) *models.Message {
	message := &models.Message{
		MessageID:   msg.Id,
		ChannelID:   msg.ChatId,
		ChannelName: chat.Title,
		SenderID:    c.getSenderID(msg),
		SenderName:  c.getSenderName(msg),
		Text:        c.getMessageText(msg),
		MediaType:   c.getMediaType(msg),
		IsForwarded: msg.ForwardInfo != nil,
		Timestamp:   time.Unix(int64(msg.Date), 0),
	}

	return message
}

// getSenderID extracts sender ID from message
func (c *Client) getSenderID(msg *client.Message) int64 {
	if msg.SenderId == nil {
		return 0
	}

	switch sender := msg.SenderId.(type) {
	case *client.MessageSenderUser:
		return sender.UserId
	case *client.MessageSenderChat:
		return sender.ChatId
	default:
		return 0
	}
}

// getSenderName attempts to get sender name (simplified)
func (c *Client) getSenderName(msg *client.Message) string {
	senderID := c.getSenderID(msg)
	if senderID == 0 {
		return "Unknown"
	}

	// In a production app, you'd cache user/chat info
	// For now, return the ID as string
	return fmt.Sprintf("User_%d", senderID)
}

// getMessageText extracts text from message content
func (c *Client) getMessageText(msg *client.Message) string {
	if msg.Content == nil {
		return ""
	}

	switch content := msg.Content.(type) {
	case *client.MessageText:
		return content.Text.Text
	case *client.MessagePhoto:
		if content.Caption != nil {
			return content.Caption.Text
		}
		return "[Photo]"
	case *client.MessageVideo:
		if content.Caption != nil {
			return content.Caption.Text
		}
		return "[Video]"
	case *client.MessageDocument:
		if content.Caption != nil {
			return content.Caption.Text
		}
		return "[Document]"
	case *client.MessageAnimation:
		if content.Caption != nil {
			return content.Caption.Text
		}
		return "[Animation]"
	case *client.MessageVoiceNote:
		if content.Caption != nil {
			return content.Caption.Text
		}
		return "[Voice Note]"
	case *client.MessageAudio:
		if content.Caption != nil {
			return content.Caption.Text
		}
		return "[Audio]"
	default:
		return fmt.Sprintf("[%T]", msg.Content)
	}
}

// getMediaType determines the media type of the message
func (c *Client) getMediaType(msg *client.Message) string {
	if msg.Content == nil {
		return "text"
	}

	switch msg.Content.(type) {
	case *client.MessageText:
		return "text"
	case *client.MessagePhoto:
		return "photo"
	case *client.MessageVideo:
		return "video"
	case *client.MessageDocument:
		return "document"
	case *client.MessageAnimation:
		return "animation"
	case *client.MessageVoiceNote:
		return "voice"
	case *client.MessageAudio:
		return "audio"
	case *client.MessageSticker:
		return "sticker"
	default:
		return "other"
	}
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Stop stops the client and closes the connection
func (c *Client) Stop() error {
	c.logger.Info("Stopping Telegram client...")
	c.cancel()

	// Close TDLib client
	_, err := c.tdClient.Close()
	if err != nil {
		return fmt.Errorf("failed to close TDLib client: %w", err)
	}

	c.connected = false
	c.logger.Info("Telegram client stopped")
	return nil
}
