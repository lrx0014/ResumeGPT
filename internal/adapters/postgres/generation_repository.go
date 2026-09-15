package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lrx0014/ResumeGPT/internal/generation"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
)

const generationColumns = `id,workspace_id,profile_id,opportunity_id,template_id,document_type,language,page_target,custom_instructions,pipeline_mode,writer,renderer,reviewer,state,stage,COALESCE(draft,''),COALESCE(rendered_source,''),COALESCE(review,''),repair_count,COALESCE(artifact_object_id,''),COALESCE(error_code,''),COALESCE(error_message,''),profile_snapshot,opportunity_snapshot,template_snapshot,created_at,updated_at`

type GenerationRepository struct{ pool *pgxpool.Pool }

func NewGenerationRepository(pool *pgxpool.Pool) *GenerationRepository {
	return &GenerationRepository{pool: pool}
}

func (r *GenerationRepository) Create(ctx context.Context, value generation.Run, task workqueue.Job) (generation.Run, error) {
	err := withWorkspaceTx(ctx, r.pool, value.WorkspaceID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO durable_jobs(id,workspace_id,kind,idempotency_key,payload,max_attempts,available_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, task.ID, task.WorkspaceID, task.Kind, task.IdempotencyKey, task.Payload, task.MaxAttempts, task.AvailableAt); err != nil {
			return err
		}
		writer, _ := json.Marshal(value.Writer)
		renderer, _ := json.Marshal(value.Renderer)
		reviewer, _ := json.Marshal(value.Reviewer)
		_, err := tx.Exec(ctx, `INSERT INTO generation_runs(id,workspace_id,profile_id,opportunity_id,template_id,document_type,language,page_target,custom_instructions,pipeline_mode,writer,renderer,reviewer,profile_snapshot,opportunity_snapshot,template_snapshot,state,stage,job_id,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`, value.ID, value.WorkspaceID, value.ProfileID, value.OpportunityID, value.TemplateID, value.DocumentType, value.Language, value.PageTarget, value.CustomInstructions, value.PipelineMode, writer, renderer, reviewer, value.ProfileSnapshot, value.OpportunitySnapshot, value.TemplateSnapshot, value.State, value.Stage, task.ID, value.CreatedAt, value.UpdatedAt)
		if err != nil {
			return err
		}
		if err := appendEvent(ctx, tx, value.WorkspaceID, "generation.queued.v1", "generation", value.ID, map[string]string{"generationId": value.ID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, value.WorkspaceID, "generation.create", "generation", value.ID)
	})
	return value, err
}
func (r *GenerationRepository) List(ctx context.Context, workspaceID string) ([]generation.Run, error) {
	items := []generation.Run{}
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT `+generationColumns+` FROM generation_runs WHERE workspace_id=$1 ORDER BY created_at DESC`, workspaceID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item generation.Run
			if err := scanGeneration(rows, &item); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}
func (r *GenerationRepository) Get(ctx context.Context, workspaceID, id string) (generation.Run, error) {
	var item generation.Run
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		err := scanGeneration(tx.QueryRow(ctx, `SELECT `+generationColumns+` FROM generation_runs WHERE workspace_id=$1 AND id=$2`, workspaceID, id), &item)
		if errors.Is(err, pgx.ErrNoRows) {
			return generation.ErrNotFound
		}
		return err
	})
	return item, err
}
func (r *GenerationRepository) Retry(ctx context.Context, workspaceID, id string) (generation.Run, error) {
	var item generation.Run
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		var jobID string
		var errorCode string
		err := tx.QueryRow(ctx, `SELECT job_id,COALESCE(error_code,'') FROM generation_runs WHERE workspace_id=$1 AND id=$2 AND state='failed' FOR UPDATE`, workspaceID, id).Scan(&jobID, &errorCode)
		if errors.Is(err, pgx.ErrNoRows) {
			var exists bool
			if lookupErr := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM generation_runs WHERE workspace_id=$1 AND id=$2)`, workspaceID, id).Scan(&exists); lookupErr != nil {
				return lookupErr
			}
			if exists {
				return generation.ErrState
			}
			return generation.ErrNotFound
		}
		if err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE durable_jobs SET state='queued',attempt=0,available_at=now(),lease_owner=NULL,lease_expires_at=NULL,heartbeat_at=NULL,error_class=NULL,error_message=NULL,payload=CASE WHEN $3 THEN jsonb_set(payload,'{useFallback}','true'::jsonb,true) ELSE payload END,updated_at=now() WHERE workspace_id=$1 AND id=$2 AND state='failed'`, workspaceID, jobID, errorCode == "fallback_render_failed")
		if err != nil || tag.RowsAffected() != 1 {
			return generation.ErrState
		}
		_, err = tx.Exec(ctx, `UPDATE generation_runs SET state='queued',stage='queued',error_code=NULL,error_message=NULL,updated_at=now() WHERE workspace_id=$1 AND id=$2`, workspaceID, id)
		if err != nil {
			return err
		}
		if err := appendEvent(ctx, tx, workspaceID, "generation.retried.v1", "generation", id, map[string]string{"generationId": id}); err != nil {
			return err
		}
		if err := appendAudit(ctx, tx, workspaceID, "generation.retry", "generation", id); err != nil {
			return err
		}
		return scanGeneration(tx.QueryRow(ctx, `SELECT `+generationColumns+` FROM generation_runs WHERE workspace_id=$1 AND id=$2`, workspaceID, id), &item)
	})
	return item, err
}
func (r *GenerationRepository) SetStage(ctx context.Context, task workqueue.Job, stage, draft, review string, repairs int) error {
	return withWorkspaceTx(ctx, r.pool, task.WorkspaceID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE generation_runs SET state='running',stage=$3,draft=CASE WHEN $4='' THEN draft ELSE $4 END,review=CASE WHEN $5='' THEN review ELSE $5 END,repair_count=$6,error_code=NULL,error_message=NULL,updated_at=now() WHERE workspace_id=$1 AND job_id=$2`, task.WorkspaceID, task.ID, stage, draft, review, repairs)
		return err
	})
}
func (r *GenerationRepository) Complete(ctx context.Context, task workqueue.Job, source, draft, review, objectID string, repairs int) error {
	return withWorkspaceTx(ctx, r.pool, task.WorkspaceID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx, `UPDATE generation_runs SET state='ready',stage='ready',rendered_source=$3,draft=$4,review=$5,artifact_object_id=$6,repair_count=$7,error_code=NULL,error_message=NULL,updated_at=now() WHERE workspace_id=$1 AND job_id=$2`, task.WorkspaceID, task.ID, source, draft, review, objectID, repairs)
		if err != nil || tag.RowsAffected() != 1 {
			return fmt.Errorf("complete generation run")
		}
		tag, err = tx.Exec(ctx, `UPDATE durable_jobs SET state='succeeded',lease_owner=NULL,lease_expires_at=NULL,updated_at=now() WHERE id=$1 AND lease_owner=$2 AND state='running'`, task.ID, task.LeaseOwner)
		if err != nil || tag.RowsAffected() != 1 {
			return fmt.Errorf("complete generation task")
		}
		if err := appendEvent(ctx, tx, task.WorkspaceID, "generation.ready.v1", "generation", task.IdempotencyKey, map[string]string{"generationId": task.IdempotencyKey}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, task.WorkspaceID, "generation.complete", "generation", task.IdempotencyKey)
	})
}
func (r *GenerationRepository) Fail(ctx context.Context, task workqueue.Job, code, message string) error {
	return withWorkspaceTx(ctx, r.pool, task.WorkspaceID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `UPDATE generation_runs SET state='failed',stage='failed',error_code=$3,error_message=$4,updated_at=now() WHERE workspace_id=$1 AND job_id=$2`, task.WorkspaceID, task.ID, code, message)
		return err
	})
}

type generationScanner interface{ Scan(...any) error }

func scanGeneration(row generationScanner, item *generation.Run) error {
	var writer, renderer, reviewer []byte
	err := row.Scan(&item.ID, &item.WorkspaceID, &item.ProfileID, &item.OpportunityID, &item.TemplateID, &item.DocumentType, &item.Language, &item.PageTarget, &item.CustomInstructions, &item.PipelineMode, &writer, &renderer, &reviewer, &item.State, &item.Stage, &item.Draft, &item.RenderedSource, &item.Review, &item.RepairCount, &item.ArtifactObjectID, &item.ErrorCode, &item.ErrorMessage, &item.ProfileSnapshot, &item.OpportunitySnapshot, &item.TemplateSnapshot, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return err
	}
	if json.Unmarshal(writer, &item.Writer) != nil || json.Unmarshal(renderer, &item.Renderer) != nil || json.Unmarshal(reviewer, &item.Reviewer) != nil {
		return fmt.Errorf("invalid generation model choices")
	}
	return nil
}
