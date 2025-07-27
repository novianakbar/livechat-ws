package domain

import "github.com/google/uuid"

type SessionConnectionEvent struct {
	SessionID uuid.UUID `json:"session_id"`
	UserID    uuid.UUID `json:"user_id"`
	UserType  string    `json:"user_type"` // agent/customer
	Action    string    `json:"action"`    // join/leave
}
