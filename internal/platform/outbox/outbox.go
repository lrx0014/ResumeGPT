package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrEmpty = errors.New("no outbox event available")

type Event struct {
	ID            string
	WorkspaceID   string
	Topic         string
	AggregateType string
	AggregateID   string
	Payload       json.RawMessage
	OccurredAt    time.Time
	Attempt       int
}

type Repository interface {
	Claim(ctx context.Context, workerID string, lease time.Duration) (Event, error)
	MarkPublished(ctx context.Context, eventID, workerID string) error
	MarkFailed(ctx context.Context, eventID, workerID, message string) error
}

type Publisher interface {
	Publish(ctx context.Context, event Event) error
}
