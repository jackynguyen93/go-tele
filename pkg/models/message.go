package models

import "time"

// Message represents a Telegram message stored in the database
type Message struct {
	ID          int64     `db:"id"`
	MessageID   int64     `db:"message_id"`
	ChannelID   int64     `db:"channel_id"`
	ChannelName string    `db:"channel_name"`
	SenderID    int64     `db:"sender_id"`
	SenderName  string    `db:"sender_name"`
	Text        string    `db:"text"`
	MediaType   string    `db:"media_type"`
	IsForwarded bool      `db:"is_forwarded"`
	Timestamp   time.Time `db:"timestamp"`
	CreatedAt   time.Time `db:"created_at"`
}

// Channel represents a subscribed Telegram channel
type Channel struct {
	ID        int64     `db:"id" json:"id"`
	ChannelID int64     `db:"channel_id" json:"channel_id"`
	Username  string    `db:"username" json:"username"`
	Title     string    `db:"title" json:"title"`
	IsActive  bool      `db:"is_active" json:"is_active"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
