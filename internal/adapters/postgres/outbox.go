package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lrx0014/ResumeGPT/internal/platform/outbox"
)

type OutboxRepository struct {
	pool *pgxpool.Pool
}

func NewOutboxRepository(pool *pgxpool.Pool) *OutboxRepository {
	return &OutboxRepository{pool: pool}
}

func (r *OutboxRepository) Claim(ctx context.Context, workerID string, lease time.Duration) (outbox.Event, error) {
	var event outbox.Event
	err := r.pool.QueryRow(ctx, "SELECT id, workspace_id, topic, aggregate_type, aggregate_id, payload, occurred_at, attempt FROM claim_outbox_event($1, $2)",
		workerID, int(lease.Seconds()),
	).Scan(&event.ID, &event.WorkspaceID, &event.Topic, &event.AggregateType, &event.AggregateID, &event.Payload, &event.OccurredAt, &event.Attempt)
	if errors.Is(err, pgx.ErrNoRows) {
		return outbox.Event{}, outbox.ErrEmpty
	}
	if err != nil {
		return outbox.Event{}, fmt.Errorf("claim outbox event: %w", err)
	}
	return event, nil
}

func (r *OutboxRepository) MarkPublished(ctx context.Context, eventID, workerID string) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE outbox_events SET published_at = now(), locked_by = NULL, locked_until = NULL, last_error = NULL
		WHERE id = $1 AND locked_by = $2 AND published_at IS NULL`, eventID, workerID)
	if err != nil || tag.RowsAffected() != 1 {
		return fmt.Errorf("mark outbox event published: lease lost")
	}
	return nil
}

func (r *OutboxRepository) MarkFailed(ctx context.Context, eventID, workerID, message string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE outbox_events
		SET locked_by = NULL,
		    locked_until = now() + make_interval(secs => LEAST(POWER(2, attempt)::integer, 300)),
		    last_error = $3
		WHERE id = $1 AND locked_by = $2 AND published_at IS NULL`, eventID, workerID, message)
	return err
}
