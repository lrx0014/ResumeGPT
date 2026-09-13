package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lrx0014/ResumeGPT/internal/profile"
)

type ProfileRepository struct {
	pool *pgxpool.Pool
}

func NewProfileRepository(pool *pgxpool.Pool) *ProfileRepository {
	return &ProfileRepository{pool: pool}
}

func (r *ProfileRepository) List(ctx context.Context, workspaceID string) ([]profile.Profile, error) {
	var items []profile.Profile
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, workspace_id, name, domain, default_language, description, created_at, updated_at
			FROM profiles
			WHERE workspace_id = $1
			ORDER BY created_at DESC, id`, workspaceID)
		if err != nil {
			return fmt.Errorf("query profiles: %w", err)
		}
		defer rows.Close()
		items = make([]profile.Profile, 0)
		for rows.Next() {
			var item profile.Profile
			if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.Name, &item.Domain, &item.DefaultLanguage, &item.Description, &item.CreatedAt, &item.UpdatedAt); err != nil {
				return fmt.Errorf("scan profile: %w", err)
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}

func (r *ProfileRepository) Get(ctx context.Context, workspaceID, profileID string) (profile.Profile, error) {
	var item profile.Profile
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `
			SELECT id, workspace_id, name, domain, default_language, description, created_at, updated_at
			FROM profiles
			WHERE workspace_id = $1 AND id = $2`, workspaceID, profileID).
			Scan(&item.ID, &item.WorkspaceID, &item.Name, &item.Domain, &item.DefaultLanguage, &item.Description, &item.CreatedAt, &item.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return profile.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("query profile: %w", err)
		}
		return nil
	})
	return item, err
}

func (r *ProfileRepository) Create(ctx context.Context, value profile.Profile) (profile.Profile, error) {
	err := withWorkspaceTx(ctx, r.pool, value.WorkspaceID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO profiles
				(id, workspace_id, name, domain, default_language, description, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			value.ID, value.WorkspaceID, value.Name, value.Domain, value.DefaultLanguage, value.Description, value.CreatedAt, value.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert profile: %w", err)
		}
		if err := appendEvent(ctx, tx, value.WorkspaceID, "profile.created.v1", "profile", value.ID, value); err != nil {
			return err
		}
		return appendAudit(ctx, tx, value.WorkspaceID, "profile.create", "profile", value.ID)
	})
	return value, err
}
