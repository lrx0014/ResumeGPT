package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lrx0014/ResumeGPT/internal/settings"
)

type SettingsRepository struct {
	pool *pgxpool.Pool
}

func NewSettingsRepository(pool *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{pool: pool}
}

func (r *SettingsRepository) GetPreferences(ctx context.Context, workspaceID string) (settings.Preferences, error) {
	value := settings.Preferences{WorkspaceID: workspaceID, InterfaceLanguage: "en", Theme: "system"}
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT workspace_id,interface_language,theme,updated_at FROM workspace_settings WHERE workspace_id=$1`, workspaceID).
			Scan(&value.WorkspaceID, &value.InterfaceLanguage, &value.Theme, &value.UpdatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	})
	return value, err
}

func (r *SettingsRepository) SavePreferences(ctx context.Context, value settings.Preferences) (settings.Preferences, error) {
	err := withWorkspaceTx(ctx, r.pool, value.WorkspaceID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO workspace_settings (workspace_id,interface_language,theme,updated_at)
			VALUES ($1,$2,$3,$4) ON CONFLICT (workspace_id) DO UPDATE SET
			interface_language=EXCLUDED.interface_language,theme=EXCLUDED.theme,updated_at=EXCLUDED.updated_at`,
			value.WorkspaceID, value.InterfaceLanguage, value.Theme, value.UpdatedAt); err != nil {
			return fmt.Errorf("save workspace settings: %w", err)
		}
		if err := appendEvent(ctx, tx, value.WorkspaceID, "settings.updated.v1", "workspace", value.WorkspaceID,
			map[string]string{"workspaceId": value.WorkspaceID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, value.WorkspaceID, "settings.update", "workspace", value.WorkspaceID)
	})
	return value, err
}

func (r *SettingsRepository) ListAgentDefaults(ctx context.Context, workspaceID string) ([]settings.AgentDefault, error) {
	result := make([]settings.AgentDefault, 0)
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT agent,connection_id,model,COALESCE(max_tokens,0),updated_at
			FROM agent_llm_defaults WHERE workspace_id=$1 ORDER BY agent`, workspaceID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var value settings.AgentDefault
			if err := rows.Scan(&value.Agent, &value.ConnectionID, &value.Model, &value.MaxTokens, &value.UpdatedAt); err != nil {
				return err
			}
			result = append(result, value)
		}
		return rows.Err()
	})
	return result, err
}

func (r *SettingsRepository) SaveAgentDefaults(ctx context.Context, workspaceID string, values []settings.AgentDefault) ([]settings.AgentDefault, error) {
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM agent_llm_defaults WHERE workspace_id=$1`, workspaceID); err != nil {
			return err
		}
		for _, value := range values {
			if _, err := tx.Exec(ctx, `INSERT INTO agent_llm_defaults (workspace_id,agent,connection_id,model,max_tokens,updated_at)
				VALUES ($1,$2,$3,$4,NULLIF($5,0),$6)`, workspaceID, value.Agent, value.ConnectionID, value.Model, value.MaxTokens, value.UpdatedAt); err != nil {
				if isSettingsConstraintError(err) {
					return settings.ErrInvalid
				}
				return err
			}
		}
		if err := appendEvent(ctx, tx, workspaceID, "settings.agent_defaults.updated.v1", "workspace", workspaceID,
			map[string]string{"workspaceId": workspaceID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, workspaceID, "settings.agent_defaults.update", "workspace", workspaceID)
	})
	return values, err
}

func (r *SettingsRepository) ListConnections(ctx context.Context, workspaceID string) ([]settings.StoredConnection, error) {
	result := make([]settings.StoredConnection, 0)
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT id,workspace_id,name,execution_mode,provider,base_url,
			api_token_ciphertext,created_at,updated_at FROM llm_connections WHERE workspace_id=$1 ORDER BY name,id`, workspaceID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item settings.StoredConnection
			if err := rows.Scan(connectionFields(&item)...); err != nil {
				return err
			}
			result = append(result, item)
		}
		return rows.Err()
	})
	return result, err
}

func (r *SettingsRepository) CountConnections(ctx context.Context, workspaceID string) (int, error) {
	var total int
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT COUNT(*) FROM llm_connections WHERE workspace_id=$1`, workspaceID).Scan(&total)
		if err != nil {
			return fmt.Errorf("count llm connections: %w", err)
		}
		return nil
	})
	return total, err
}

func (r *SettingsRepository) GetConnection(ctx context.Context, workspaceID, connectionID string) (settings.StoredConnection, error) {
	var result settings.StoredConnection
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT id,workspace_id,name,execution_mode,provider,base_url,
			api_token_ciphertext,created_at,updated_at FROM llm_connections WHERE workspace_id=$1 AND id=$2`, workspaceID, connectionID).
			Scan(connectionFields(&result)...)
		if errors.Is(err, pgx.ErrNoRows) {
			return settings.ErrNotFound
		}
		return err
	})
	return result, err
}

func (r *SettingsRepository) CreateConnection(ctx context.Context, value settings.StoredConnection) (settings.StoredConnection, error) {
	err := withWorkspaceTx(ctx, r.pool, value.Connection.WorkspaceID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO llm_connections
			(id,workspace_id,name,execution_mode,provider,base_url,api_token_ciphertext,created_at,updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, value.Connection.ID, value.Connection.WorkspaceID,
			value.Connection.Name, value.Connection.ExecutionMode, value.Connection.Provider, value.Connection.BaseURL,
			nullableBytes(value.TokenCiphertext), value.Connection.CreatedAt, value.Connection.UpdatedAt); err != nil {
			if isSettingsConstraintError(err) {
				return settings.ErrInvalid
			}
			return fmt.Errorf("create LLM provider: %w", err)
		}
		if err := appendEvent(ctx, tx, value.Connection.WorkspaceID, "llm.connection.created.v1", "llm_connection", value.Connection.ID,
			map[string]string{"connectionId": value.Connection.ID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, value.Connection.WorkspaceID, "llm.connection.create", "llm_connection", value.Connection.ID)
	})
	return value, err
}

func (r *SettingsRepository) UpdateConnection(ctx context.Context, value settings.StoredConnection) (settings.StoredConnection, error) {
	err := withWorkspaceTx(ctx, r.pool, value.Connection.WorkspaceID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE llm_connections SET name=$3,execution_mode=$4,provider=$5,base_url=$6,
			api_token_ciphertext=$7,updated_at=$8 WHERE workspace_id=$1 AND id=$2`,
			value.Connection.WorkspaceID, value.Connection.ID, value.Connection.Name, value.Connection.ExecutionMode,
			value.Connection.Provider, value.Connection.BaseURL, nullableBytes(value.TokenCiphertext), value.Connection.UpdatedAt)
		if err != nil {
			if isSettingsConstraintError(err) {
				return settings.ErrInvalid
			}
			return err
		}
		if tag.RowsAffected() == 0 {
			return settings.ErrNotFound
		}
		if err := appendEvent(ctx, tx, value.Connection.WorkspaceID, "llm.connection.updated.v1", "llm_connection", value.Connection.ID,
			map[string]string{"connectionId": value.Connection.ID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, value.Connection.WorkspaceID, "llm.connection.update", "llm_connection", value.Connection.ID)
	})
	return value, err
}

func (r *SettingsRepository) DeleteConnection(ctx context.Context, workspaceID, connectionID string) error {
	return withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `DELETE FROM llm_connections WHERE workspace_id=$1 AND id=$2`, workspaceID, connectionID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return settings.ErrNotFound
		}
		if err := appendEvent(ctx, tx, workspaceID, "llm.connection.deleted.v1", "llm_connection", connectionID,
			map[string]string{"connectionId": connectionID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, workspaceID, "llm.connection.delete", "llm_connection", connectionID)
	})
}

func connectionFields(value *settings.StoredConnection) []any {
	return []any{&value.Connection.ID, &value.Connection.WorkspaceID, &value.Connection.Name,
		&value.Connection.ExecutionMode, &value.Connection.Provider, &value.Connection.BaseURL,
		&value.TokenCiphertext, &value.Connection.CreatedAt, &value.Connection.UpdatedAt}
}

func nullableBytes(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func isSettingsConstraintError(err error) bool {
	var databaseError *pgconn.PgError
	return errors.As(err, &databaseError) && (databaseError.Code == "23503" || databaseError.Code == "23505" || databaseError.Code == "23514")
}
