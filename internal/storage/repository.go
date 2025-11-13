package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"tdlib-go/pkg/models"
)

// Repository handles database operations
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new repository instance
func NewRepository(dsn string) (*Repository, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrent performance
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	repo := &Repository{db: db}

	// Run migrations
	if err := repo.migrate(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return repo, nil
}

// migrate runs database migrations
func (r *Repository) migrate() error {
	schema, err := os.ReadFile("migrations/001_initial_schema.sql")
	if err != nil {
		// If migrations file doesn't exist, create schema inline
		return r.createSchemaInline()
	}

	if _, err := r.db.Exec(string(schema)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	return nil
}

// createSchemaInline creates the database schema inline
func (r *Repository) createSchemaInline() error {
	schema := `
	CREATE TABLE IF NOT EXISTS channels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		channel_id INTEGER NOT NULL UNIQUE,
		username TEXT,
		title TEXT NOT NULL,
		is_active BOOLEAN DEFAULT 1,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_channels_channel_id ON channels(channel_id);
	CREATE INDEX IF NOT EXISTS idx_channels_username ON channels(username);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		message_id INTEGER NOT NULL,
		channel_id INTEGER NOT NULL,
		channel_name TEXT NOT NULL,
		sender_id INTEGER NOT NULL,
		sender_name TEXT,
		text TEXT,
		media_type TEXT,
		is_forwarded BOOLEAN DEFAULT 0,
		timestamp TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(message_id, channel_id)
	);

	CREATE INDEX IF NOT EXISTS idx_messages_channel_id ON messages(channel_id);
	CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
	CREATE INDEX IF NOT EXISTS idx_messages_sender_id ON messages(sender_id);
	`

	if _, err := r.db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// Close closes the database connection
func (r *Repository) Close() error {
	return r.db.Close()
}

// SaveMessage saves a message to the database
func (r *Repository) SaveMessage(msg *models.Message) error {
	query := `
		INSERT OR IGNORE INTO messages
		(message_id, channel_id, channel_name, sender_id, sender_name, text, media_type, is_forwarded, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Exec(query,
		msg.MessageID,
		msg.ChannelID,
		msg.ChannelName,
		msg.SenderID,
		msg.SenderName,
		msg.Text,
		msg.MediaType,
		msg.IsForwarded,
		msg.Timestamp,
	)

	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	return nil
}

// GetMessagesByChannel retrieves messages from a specific channel
func (r *Repository) GetMessagesByChannel(channelID int64, limit int) ([]*models.Message, error) {
	query := `
		SELECT id, message_id, channel_id, channel_name, sender_id, sender_name,
		       text, media_type, is_forwarded, timestamp, created_at
		FROM messages
		WHERE channel_id = ?
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := r.db.Query(query, channelID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer rows.Close()

	var messages []*models.Message
	for rows.Next() {
		msg := &models.Message{}
		err := rows.Scan(
			&msg.ID,
			&msg.MessageID,
			&msg.ChannelID,
			&msg.ChannelName,
			&msg.SenderID,
			&msg.SenderName,
			&msg.Text,
			&msg.MediaType,
			&msg.IsForwarded,
			&msg.Timestamp,
			&msg.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// SaveChannel saves a channel to the database
func (r *Repository) SaveChannel(channel *models.Channel) error {
	query := `
		INSERT INTO channels (channel_id, username, title, is_active)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(channel_id) DO UPDATE SET
			username = excluded.username,
			title = excluded.title,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.Exec(query,
		channel.ChannelID,
		channel.Username,
		channel.Title,
		channel.IsActive,
	)

	if err != nil {
		return fmt.Errorf("failed to save channel: %w", err)
	}

	return nil
}

// GetChannel retrieves a channel by ID or username
func (r *Repository) GetChannel(identifier string) (*models.Channel, error) {
	query := `
		SELECT id, channel_id, username, title, is_active, created_at, updated_at
		FROM channels
		WHERE channel_id = ? OR username = ?
		LIMIT 1
	`

	channel := &models.Channel{}
	err := r.db.QueryRow(query, identifier, identifier).Scan(
		&channel.ID,
		&channel.ChannelID,
		&channel.Username,
		&channel.Title,
		&channel.IsActive,
		&channel.CreatedAt,
		&channel.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	return channel, nil
}

// GetAllChannels retrieves all active channels
func (r *Repository) GetAllChannels() ([]*models.Channel, error) {
	query := `
		SELECT id, channel_id, username, title, is_active, created_at, updated_at
		FROM channels
		WHERE is_active = 1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query channels: %w", err)
	}
	defer rows.Close()

	var channels []*models.Channel
	for rows.Next() {
		channel := &models.Channel{}
		err := rows.Scan(
			&channel.ID,
			&channel.ChannelID,
			&channel.Username,
			&channel.Title,
			&channel.IsActive,
			&channel.CreatedAt,
			&channel.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan channel: %w", err)
		}
		channels = append(channels, channel)
	}

	return channels, nil
}

// DeactivateChannel marks a channel as inactive
func (r *Repository) DeactivateChannel(channelID int64) error {
	query := `UPDATE channels SET is_active = 0, updated_at = ? WHERE channel_id = ?`
	_, err := r.db.Exec(query, time.Now(), channelID)
	if err != nil {
		return fmt.Errorf("failed to deactivate channel: %w", err)
	}
	return nil
}

// GetDatabasePath returns the full path to the database file
func GetDatabasePath(dsn string) (string, error) {
	dir := filepath.Dir(dsn)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", fmt.Errorf("failed to create database directory: %w", err)
		}
	}
	return dsn, nil
}
