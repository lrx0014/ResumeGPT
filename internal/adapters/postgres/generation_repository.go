package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lrx0014/ResumeGPT/internal/generation"
	"github.com/lrx0014/ResumeGPT/internal/platform/workqueue"
	"github.com/lrx0014/ResumeGPT/internal/shared/id"
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
func (r *GenerationRepository) Delete(ctx context.Context, workspaceID, generationID string) error {
	return withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		var jobID string
		err := tx.QueryRow(ctx, `SELECT job_id FROM generation_runs WHERE workspace_id=$1 AND id=$2 FOR UPDATE`, workspaceID, generationID).Scan(&jobID)
		if errors.Is(err, pgx.ErrNoRows) {
			return generation.ErrNotFound
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM generation_runs WHERE workspace_id=$1 AND id=$2`, workspaceID, generationID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM durable_jobs WHERE workspace_id=$1 AND id=$2`, workspaceID, jobID); err != nil {
			return err
		}
		return appendAudit(ctx, tx, workspaceID, "generation.delete", "generation", generationID)
	})
}
func (r *GenerationRepository) Reconfigure(ctx context.Context, value generation.Run) (generation.Run, error) {
	var item generation.Run
	err := withWorkspaceTx(ctx, r.pool, value.WorkspaceID, func(tx pgx.Tx) error {
		var jobID string
		err := tx.QueryRow(ctx, `SELECT job_id FROM generation_runs WHERE workspace_id=$1 AND id=$2 AND state IN ('ready','failed') FOR UPDATE`, value.WorkspaceID, value.ID).Scan(&jobID)
		if errors.Is(err, pgx.ErrNoRows) {
			var exists bool
			if lookupErr := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM generation_runs WHERE workspace_id=$1 AND id=$2)`, value.WorkspaceID, value.ID).Scan(&exists); lookupErr != nil {
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
		tag, err := tx.Exec(ctx, `UPDATE durable_jobs SET state='queued',attempt=0,available_at=now(),lease_owner=NULL,lease_expires_at=NULL,heartbeat_at=NULL,error_class=NULL,error_message=NULL,payload=jsonb_build_object('runId',$3::text),updated_at=now() WHERE workspace_id=$1 AND id=$2 AND state IN ('succeeded','failed','retry_wait')`, value.WorkspaceID, jobID, value.ID)
		if err != nil || tag.RowsAffected() != 1 {
			return generation.ErrState
		}
		writer, _ := json.Marshal(value.Writer)
		renderer, _ := json.Marshal(value.Renderer)
		reviewer, _ := json.Marshal(value.Reviewer)
		_, err = tx.Exec(ctx, `UPDATE generation_runs SET profile_id=$3,opportunity_id=$4,template_id=$5,document_type=$6,language=$7,page_target=$8,custom_instructions=$9,pipeline_mode=$10,writer=$11,renderer=$12,reviewer=$13,profile_snapshot=$14,opportunity_snapshot=$15,template_snapshot=$16,state='queued',stage='queued',draft=NULL,rendered_source=NULL,review=NULL,repair_count=0,artifact_object_id=NULL,error_code=NULL,error_message=NULL,updated_at=$17 WHERE workspace_id=$1 AND id=$2`, value.WorkspaceID, value.ID, value.ProfileID, value.OpportunityID, value.TemplateID, value.DocumentType, value.Language, value.PageTarget, value.CustomInstructions, value.PipelineMode, writer, renderer, reviewer, value.ProfileSnapshot, value.OpportunitySnapshot, value.TemplateSnapshot, value.UpdatedAt)
		if err != nil {
			return err
		}
		var sequence int
		if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(sequence),0)+1 FROM generation_steps WHERE workspace_id=$1 AND generation_id=$2`, value.WorkspaceID, value.ID).Scan(&sequence); err != nil {
			return err
		}
		configuration, _ := json.Marshal(map[string]any{"profileId": value.ProfileID, "templateId": value.TemplateID, "pipelineMode": value.PipelineMode, "writer": value.Writer, "renderer": value.Renderer, "reviewer": value.Reviewer})
		if _, err := tx.Exec(ctx, `INSERT INTO generation_steps(id,workspace_id,generation_id,kind,sequence,content,feedback,created_at) VALUES($1,$2,$3,'configuration_change',$4,$5,$6,$7)`, id.New("step"), value.WorkspaceID, value.ID, sequence, string(configuration), "Configuration updated. A fresh generation was queued.", value.UpdatedAt); err != nil {
			return err
		}
		if err := appendEvent(ctx, tx, value.WorkspaceID, "generation.reconfigured.v1", "generation", value.ID, map[string]string{"generationId": value.ID}); err != nil {
			return err
		}
		if err := appendAudit(ctx, tx, value.WorkspaceID, "generation.reconfigure", "generation", value.ID); err != nil {
			return err
		}
		return scanGeneration(tx.QueryRow(ctx, `SELECT `+generationColumns+` FROM generation_runs WHERE workspace_id=$1 AND id=$2`, value.WorkspaceID, value.ID), &item)
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
func (r *GenerationRepository) Revise(ctx context.Context, workspaceID, generationID, prompt string) (generation.Run, error) {
	var item generation.Run
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		var jobID string
		err := tx.QueryRow(ctx, `SELECT job_id FROM generation_runs WHERE workspace_id=$1 AND id=$2 AND state='ready' FOR UPDATE`, workspaceID, generationID).Scan(&jobID)
		if errors.Is(err, pgx.ErrNoRows) {
			var exists bool
			if lookupErr := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM generation_runs WHERE workspace_id=$1 AND id=$2)`, workspaceID, generationID).Scan(&exists); lookupErr != nil {
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
		var sequence int
		if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(sequence),0)+1 FROM generation_steps WHERE workspace_id=$1 AND generation_id=$2`, workspaceID, generationID).Scan(&sequence); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO generation_steps(id,workspace_id,generation_id,kind,sequence,content,created_at) VALUES($1,$2,$3,'user_prompt',$4,$5,$6)`, id.New("step"), workspaceID, generationID, sequence, prompt, time.Now().UTC()); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `UPDATE durable_jobs SET state='queued',attempt=0,available_at=now(),lease_owner=NULL,lease_expires_at=NULL,heartbeat_at=NULL,error_class=NULL,error_message=NULL,payload=jsonb_set(jsonb_set(payload,'{useFallback}','false'::jsonb,true),'{revisionPrompt}',to_jsonb($3::text),true),updated_at=now() WHERE workspace_id=$1 AND id=$2 AND state='succeeded'`, workspaceID, jobID, prompt)
		if err != nil || tag.RowsAffected() != 1 {
			return generation.ErrState
		}
		if _, err := tx.Exec(ctx, `UPDATE generation_runs SET state='queued',stage='queued',error_code=NULL,error_message=NULL,updated_at=now() WHERE workspace_id=$1 AND id=$2`, workspaceID, generationID); err != nil {
			return err
		}
		if err := appendEvent(ctx, tx, workspaceID, "generation.revision_queued.v1", "generation", generationID, map[string]string{"generationId": generationID}); err != nil {
			return err
		}
		if err := appendAudit(ctx, tx, workspaceID, "generation.revise", "generation", generationID); err != nil {
			return err
		}
		return scanGeneration(tx.QueryRow(ctx, `SELECT `+generationColumns+` FROM generation_runs WHERE workspace_id=$1 AND id=$2`, workspaceID, generationID), &item)
	})
	return item, err
}
func (r *GenerationRepository) RecordStep(ctx context.Context, task workqueue.Job, step generation.Step) error {
	return withWorkspaceTx(ctx, r.pool, task.WorkspaceID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `SELECT id FROM generation_runs WHERE workspace_id=$1 AND job_id=$2 FOR UPDATE`, task.WorkspaceID, task.ID); err != nil {
			return err
		}
		var sequence int
		if err := tx.QueryRow(ctx, `SELECT COALESCE(MAX(sequence),0)+1 FROM generation_steps WHERE workspace_id=$1 AND generation_id=$2`, task.WorkspaceID, task.IdempotencyKey).Scan(&sequence); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO generation_steps(id,workspace_id,generation_id,kind,sequence,content,feedback,artifact_object_id,repair_count,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,''),$9,$10)`, step.ID, task.WorkspaceID, task.IdempotencyKey, step.Kind, sequence, step.Content, step.Feedback, step.ArtifactObjectID, step.RepairCount, step.CreatedAt)
		return err
	})
}
func (r *GenerationRepository) ListSteps(ctx context.Context, workspaceID, generationID string) ([]generation.Step, error) {
	items := []generation.Step{}
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `SELECT id,workspace_id,generation_id,kind,sequence,content,feedback,COALESCE(artifact_object_id,''),repair_count,created_at FROM generation_steps WHERE workspace_id=$1 AND generation_id=$2 ORDER BY sequence`, workspaceID, generationID)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item generation.Step
			if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.GenerationID, &item.Kind, &item.Sequence, &item.Content, &item.Feedback, &item.ArtifactObjectID, &item.RepairCount, &item.CreatedAt); err != nil {
				return err
			}
			items = append(items, item)
		}
		return rows.Err()
	})
	return items, err
}
func (r *GenerationRepository) GetStep(ctx context.Context, workspaceID, generationID, stepID string) (generation.Step, error) {
	var item generation.Step
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `SELECT id,workspace_id,generation_id,kind,sequence,content,feedback,COALESCE(artifact_object_id,''),repair_count,created_at FROM generation_steps WHERE workspace_id=$1 AND generation_id=$2 AND id=$3`, workspaceID, generationID, stepID).Scan(&item.ID, &item.WorkspaceID, &item.GenerationID, &item.Kind, &item.Sequence, &item.Content, &item.Feedback, &item.ArtifactObjectID, &item.RepairCount, &item.CreatedAt)
		if errors.Is(err, pgx.ErrNoRows) {
			return generation.ErrNotFound
		}
		return err
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
