package domain

import (
	"time"

	"github.com/google/uuid"
)

type ChatMessage struct {
	ID          uuid.UUID  `json:"id"`
	SessionID   uuid.UUID  `json:"session_id"`
	SenderID    *uuid.UUID `json:"sender_id"`
	SenderType  string     `json:"sender_type"`
	Message     string     `json:"message"`
	MessageType string     `json:"message_type"`
	Attachments []string   `json:"attachments"`
	ReadAt      *time.Time `json:"read_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type TypingMessage struct {
	Type      string    `json:"type"`
	SessionID uuid.UUID `json:"session_id"`
	UserID    string    `json:"user_id"`
	UserType  string    `json:"user_type"`
	IsTyping  bool      `json:"is_typing"`
	Timestamp time.Time `json:"timestamp"`
}

type OnlineStatusMessage struct {
	Type      string    `json:"type"`
	SessionID uuid.UUID `json:"session_id"`
	UserID    string    `json:"user_id"`
	UserType  string    `json:"user_type"`
	IsOnline  bool      `json:"is_online"`
	Timestamp time.Time `json:"timestamp"`
}

type ConnectionStatusMessage struct {
	Type             string                 `json:"type"`
	SessionID        uuid.UUID              `json:"session_id"`
	ConnectionStatus map[string]interface{} `json:"connection_status"`
	Timestamp        time.Time              `json:"timestamp"`
}
