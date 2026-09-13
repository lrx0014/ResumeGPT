package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lrx0014/ResumeGPT/internal/platform/requestcontext"
	"github.com/lrx0014/ResumeGPT/internal/shared/id"
)

func withWorkspaceTx(ctx context.Context, pool *pgxpool.Pool, workspaceID string, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin workspace transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SELECT set_config('app.workspace_id', $1, true)", workspaceID); err != nil {
		return fmt.Errorf("set workspace scope: %w", err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit workspace transaction: %w", err)
	}
	return nil
}

func appendEvent(ctx context.Context, tx pgx.Tx, workspaceID, topic, aggregateType, aggregateID string, payload any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode outbox payload: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO outbox_events
			(id, workspace_id, topic, aggregate_type, aggregate_id, payload)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		id.New("evt"), workspaceID, topic, aggregateType, aggregateID, encoded,
	); err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

func appendAudit(ctx context.Context, tx pgx.Tx, workspaceID, action, resourceType, resourceID string) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events
			(id, workspace_id, actor_id, action, resource_type, resource_id, result, request_id)
		VALUES ($1, $2, $3, $4, $5, $6, 'success', $7)`,
		id.New("audit"), workspaceID, requestcontext.ActorID(ctx), action, resourceType, resourceID, requestcontext.RequestID(ctx),
	); err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}
