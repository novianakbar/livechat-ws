package domain

import (
	"time"

	"github.com/google/uuid"
)

type SendMessageRequest struct {
	SessionID   uuid.UUID `json:"session_id"`
	Message     string    `json:"message"`
	MessageType string    `json:"message_type"`
	Attachments []string  `json:"attachments"`
}

type SendMessageResponse struct {
	MessageID uuid.UUID `json:"message_id"`
	Timestamp time.Time `json:"timestamp"`
	Status    string    `json:"status"`
}

type WebSocketMessage struct {
	Type      string      `json:"type"`
	SessionID uuid.UUID   `json:"session_id"`
	UserID    string      `json:"user_id"`
	UserType  string      `json:"user_type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

type WebSocketResponse struct {
	Type    string      `json:"type"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
}

type TypingRequest struct {
	IsTyping bool `json:"is_typing"`
}

type ConnectionStatusResponse struct {
	CustomerConnected bool `json:"customer_connected"`
	AgentConnected    bool `json:"agent_connected"`
	TotalCustomer     int  `json:"total_customer"`
	TotalAgent        int  `json:"total_agent"`
}
