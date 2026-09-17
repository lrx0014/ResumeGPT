package taskmonitor

import (
	"encoding/json"
	"time"
)

type Task struct {
	ID             string          `json:"id"`
	WorkspaceID    string          `json:"workspaceId"`
	Kind           string          `json:"kind"`
	State          string          `json:"state"`
	IdempotencyKey string          `json:"idempotencyKey"`
	Payload        json.RawMessage `json:"-"`
	SafePayload    any             `json:"payload"`
	Attempt        int             `json:"attempt"`
	MaxAttempts    int             `json:"maxAttempts"`
	AvailableAt    time.Time       `json:"availableAt"`
	DeadlineAt     *time.Time      `json:"deadlineAt,omitempty"`
	LeaseExpiresAt *time.Time      `json:"leaseExpiresAt,omitempty"`
	HeartbeatAt    *time.Time      `json:"heartbeatAt,omitempty"`
	ErrorClass     string          `json:"errorClass,omitempty"`
	ErrorMessage   string          `json:"errorMessage,omitempty"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
	Events         []Event         `json:"events,omitempty"`
}

type Event struct {
	ID         int64     `json:"id"`
	EventType  string    `json:"eventType"`
	State      string    `json:"state"`
	Attempt    int       `json:"attempt"`
	ErrorClass string    `json:"errorClass,omitempty"`
	Message    string    `json:"message"`
	OccurredAt time.Time `json:"occurredAt"`
}

type Filter struct {
	State    string
	Kind     string
	Search   string
	Page     int
	PageSize int
}

type Page struct {
	Items    []Task         `json:"items"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
	Counts   map[string]int `json:"counts"`
}
