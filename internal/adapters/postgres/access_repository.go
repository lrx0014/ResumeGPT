package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lrx0014/ResumeGPT/internal/identity"
)

type AccessRepository struct {
	pool *pgxpool.Pool
}

func NewAccessRepository(pool *pgxpool.Pool) *AccessRepository {
	return &AccessRepository{pool: pool}
}

func (r *AccessRepository) Resolve(ctx context.Context, subject identity.Subject, workspaceID string) (identity.Principal, error) {
	if workspaceID == "" {
		return identity.Principal{}, identity.ErrForbidden
	}
	var principal identity.Principal
	err := r.pool.QueryRow(ctx, `
		SELECT u.id, wm.workspace_id, wm.role
		FROM users u
		JOIN workspace_memberships wm ON wm.user_id = u.id
		WHERE u.issuer = $1 AND u.subject = $2 AND wm.workspace_id = $3`,
		subject.Issuer, subject.Subject, workspaceID,
	).Scan(&principal.UserID, &principal.WorkspaceID, &principal.Role)
	if errors.Is(err, pgx.ErrNoRows) {
		return identity.Principal{}, identity.ErrForbidden
	}
	if err != nil {
		return identity.Principal{}, fmt.Errorf("resolve workspace access: %w", err)
	}
	return principal, nil
}
