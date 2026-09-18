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
			SELECT id, workspace_id, name, target_role, default_language, '', COALESCE(avatar_object_id, ''), created_at, updated_at,
				LEFT(content, 140), LENGTH(BTRIM(content)) > 0
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
			if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.Name, &item.TargetRole, &item.DefaultLanguage, &item.Content, &item.AvatarObjectID, &item.CreatedAt, &item.UpdatedAt, &item.ContentPreview, &item.HasContent); err != nil {
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
			SELECT id, workspace_id, name, target_role, default_language, content, COALESCE(avatar_object_id, ''), created_at, updated_at
			FROM profiles
			WHERE workspace_id = $1 AND id = $2`, workspaceID, profileID).
			Scan(&item.ID, &item.WorkspaceID, &item.Name, &item.TargetRole, &item.DefaultLanguage, &item.Content, &item.AvatarObjectID, &item.CreatedAt, &item.UpdatedAt)
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
				(id, workspace_id, name, target_role, default_language, content, avatar_object_id, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8, $9)`,
			value.ID, value.WorkspaceID, value.Name, value.TargetRole, value.DefaultLanguage, value.Content, value.AvatarObjectID, value.CreatedAt, value.UpdatedAt,
		); err != nil {
			return fmt.Errorf("insert profile: %w", err)
		}
		if err := appendEvent(ctx, tx, value.WorkspaceID, "profile.created.v1", "profile", value.ID, map[string]string{"profileId": value.ID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, value.WorkspaceID, "profile.create", "profile", value.ID)
	})
	return value, err
}

func (r *ProfileRepository) Update(ctx context.Context, value profile.Profile) (profile.Profile, error) {
	err := withWorkspaceTx(ctx, r.pool, value.WorkspaceID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `UPDATE profiles
			SET name=$3,target_role=$4,default_language=$5,content=$6,avatar_object_id=NULLIF($7,''),updated_at=$8
			WHERE workspace_id=$1 AND id=$2
			RETURNING created_at`, value.WorkspaceID, value.ID, value.Name, value.TargetRole, value.DefaultLanguage,
			value.Content, value.AvatarObjectID, value.UpdatedAt).Scan(&value.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return profile.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("update profile: %w", err)
		}
		if err := appendEvent(ctx, tx, value.WorkspaceID, "profile.updated.v1", "profile", value.ID, map[string]string{"profileId": value.ID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, value.WorkspaceID, "profile.update", "profile", value.ID)
	})
	return value, err
}

func (r *ProfileRepository) Delete(ctx context.Context, workspaceID, profileID string) error {
	return withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE durable_jobs SET state='cancelled',lease_owner=NULL,lease_expires_at=NULL,
			error_class='profile_deleted',error_message='The profile was deleted before document extraction started.',updated_at=now()
			WHERE workspace_id=$1 AND state IN ('queued','retry_wait') AND id IN (
				SELECT job_id FROM document_uploads WHERE workspace_id=$1 AND profile_id=$2 AND job_id IS NOT NULL
			)`, workspaceID, profileID); err != nil {
			return fmt.Errorf("cancel profile document jobs: %w", err)
		}
		tag, err := tx.Exec(ctx, `DELETE FROM profiles WHERE workspace_id=$1 AND id=$2`, workspaceID, profileID)
		if err != nil {
			return fmt.Errorf("delete profile: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return profile.ErrNotFound
		}
		if err := appendEvent(ctx, tx, workspaceID, "profile.deleted.v1", "profile", profileID, map[string]string{"profileId": profileID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, workspaceID, "profile.delete", "profile", profileID)
	})
}
