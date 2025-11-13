package telegram

import (
	"fmt"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/zelenin/go-tdlib/client"
	"tdlib-go/internal/config"
	"tdlib-go/internal/storage"
	"tdlib-go/pkg/models"
)

// Monitor manages channel subscriptions and message monitoring
type Monitor struct {
	client     *Client
	repo       *storage.Repository
	config     *config.Config
	logger     *logrus.Logger
	channels   map[int64]*models.Channel
	channelsMu sync.RWMutex
}

// NewMonitor creates a new channel monitor
func NewMonitor(client *Client, repo *storage.Repository, cfg *config.Config, logger *logrus.Logger) *Monitor {
	return &Monitor{
		client:   client,
		repo:     repo,
		config:   cfg,
		logger:   logger,
		channels: make(map[int64]*models.Channel),
	}
}

// Start initializes the monitor and subscribes to configured channels
func (m *Monitor) Start() error {
	m.logger.Info("Starting channel monitor...")

	// Register message handler
	m.client.AddMessageHandler(m.handleMessage)

	// Subscribe to channels from config
	for _, channelIdentifier := range m.config.Channels {
		if err := m.SubscribeChannel(channelIdentifier); err != nil {
			m.logger.Errorf("Failed to subscribe to channel %s: %v", channelIdentifier, err)
			continue
		}
	}

	// Load existing channels from database
	dbChannels, err := m.repo.GetAllChannels()
	if err != nil {
		m.logger.Errorf("Failed to load channels from database: %v", err)
	} else {
		for _, ch := range dbChannels {
			m.channelsMu.Lock()
			m.channels[ch.ChannelID] = ch
			m.channelsMu.Unlock()
		}
	}

	m.logger.Infof("Monitoring %d channels", len(m.channels))

	return nil
}

// SubscribeChannel subscribes to a channel and optionally fetches history
func (m *Monitor) SubscribeChannel(identifier string) error {
	m.logger.Infof("Subscribing to channel: %s", identifier)

	// Join the channel
	chat, err := m.client.JoinChat(identifier)
	if err != nil {
		return fmt.Errorf("failed to join channel: %w", err)
	}

	// Create channel model
	channel := &models.Channel{
		ChannelID: chat.Id,
		Username:  identifier,
		Title:     chat.Title,
		IsActive:  true,
	}

	// Save to database
	if err := m.repo.SaveChannel(channel); err != nil {
		return fmt.Errorf("failed to save channel: %w", err)
	}

	// Add to memory
	m.channelsMu.Lock()
	m.channels[chat.Id] = channel
	m.channelsMu.Unlock()

	m.logger.Infof("Successfully subscribed to channel: %s (ID: %d)", chat.Title, chat.Id)

	return nil
}

// FetchHistory fetches historical messages from a channel
func (m *Monitor) FetchHistory(channelID int64, limit int32) error {
	m.logger.Infof("Fetching history for channel %d (limit: %d)", channelID, limit)

	messages, err := m.client.GetChatHistory(channelID, limit)
	if err != nil {
		return fmt.Errorf("failed to fetch history: %w", err)
	}

	m.logger.Infof("Fetched %d messages from channel %d", len(messages), channelID)

	// Get chat info
	chat, err := m.client.tdClient.GetChat(&client.GetChatRequest{ChatId: channelID})
	if err != nil {
		return fmt.Errorf("failed to get chat info: %w", err)
	}

	// Process and save messages
	for _, msg := range messages {
		modelMsg := m.client.convertMessage(msg, chat)
		if err := m.repo.SaveMessage(modelMsg); err != nil {
			m.logger.Errorf("Failed to save historical message: %v", err)
		}
	}

	return nil
}

// UnsubscribeChannel unsubscribes from a channel
func (m *Monitor) UnsubscribeChannel(channelID int64) error {
	m.logger.Infof("Unsubscribing from channel: %d", channelID)

	// Deactivate in database
	if err := m.repo.DeactivateChannel(channelID); err != nil {
		return fmt.Errorf("failed to deactivate channel: %w", err)
	}

	// Remove from memory
	m.channelsMu.Lock()
	delete(m.channels, channelID)
	m.channelsMu.Unlock()

	m.logger.Infof("Successfully unsubscribed from channel: %d", channelID)

	return nil
}

// ListChannels returns all monitored channels
func (m *Monitor) ListChannels() []*models.Channel {
	m.channelsMu.RLock()
	defer m.channelsMu.RUnlock()

	channels := make([]*models.Channel, 0, len(m.channels))
	for _, ch := range m.channels {
		channels = append(channels, ch)
	}

	return channels
}

// handleMessage is called for each new message
func (m *Monitor) handleMessage(msg *models.Message) error {
	// Check if we're monitoring this channel
	m.channelsMu.RLock()
	channel, exists := m.channels[msg.ChannelID]
	m.channelsMu.RUnlock()

	if !exists {
		// Not monitoring this channel
		return nil
	}

	// Log the message
	m.logger.WithFields(logrus.Fields{
		"channel":    channel.Title,
		"channel_id": msg.ChannelID,
		"message_id": msg.MessageID,
		"sender_id":  msg.SenderID,
		"media_type": msg.MediaType,
		"forwarded":  msg.IsForwarded,
	}).Info("New message received")

	// Print to console
	fmt.Printf("\n[%s] %s (Channel: %s)\n",
		msg.Timestamp.Format("2006-01-02 15:04:05"),
		msg.SenderName,
		msg.ChannelName,
	)
	fmt.Printf("  Type: %s | Forwarded: %v\n", msg.MediaType, msg.IsForwarded)
	if msg.Text != "" {
		fmt.Printf("  Message: %s\n", msg.Text)
	}
	fmt.Println("---")

	// Save to database
	if err := m.repo.SaveMessage(msg); err != nil {
		m.logger.Errorf("Failed to save message: %v", err)
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}
