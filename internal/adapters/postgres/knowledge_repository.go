package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/lrx0014/ResumeGPT/internal/knowledge"
	"github.com/lrx0014/ResumeGPT/internal/platform/requestcontext"
	"github.com/lrx0014/ResumeGPT/internal/shared/id"
)

type KnowledgeRepository struct{ pool *pgxpool.Pool }

func NewKnowledgeRepository(pool *pgxpool.Pool) *KnowledgeRepository {
	return &KnowledgeRepository{pool: pool}
}

func knowledgeProfile(ctx context.Context, tx pgx.Tx, workspaceID, profileID string) error {
	var found string
	err := tx.QueryRow(ctx, "SELECT id FROM profiles WHERE workspace_id=$1 AND id=$2 FOR KEY SHARE", workspaceID, profileID).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return knowledge.ErrNotFound
	}
	return err
}

func (r *KnowledgeRepository) Import(ctx context.Context, workspaceID, profileID string, source knowledge.Source, segments []knowledge.Segment, facts []knowledge.Fact) (knowledge.Source, error) {
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		if err := knowledgeProfile(ctx, tx, workspaceID, profileID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `INSERT INTO profile_sources (workspace_id, profile_id, id, name, text, hash, created_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (workspace_id, profile_id, hash) DO NOTHING`, workspaceID, profileID, source.ID, source.Name, source.Text, source.Hash, source.CreatedAt)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return tx.QueryRow(ctx, `SELECT id,name,text,hash,media_type,parser_version,state,created_at FROM profile_sources WHERE workspace_id=$1 AND profile_id=$2 AND hash=$3`, workspaceID, profileID, source.Hash).
				Scan(&source.ID, &source.Name, &source.Text, &source.Hash, &source.MediaType, &source.ParserVersion, &source.State, &source.CreatedAt)
		}
		createdSourceID := source.ID
		for i, segment := range segments {
			if _, err := tx.Exec(ctx, `INSERT INTO source_segments (workspace_id,profile_id,id,source_id,page_number,paragraph,text,hash,confidence,bounding_box) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, workspaceID, profileID, segment.ID, source.ID, segment.Page, segment.Paragraph, segment.Text, segment.Hash, segment.Confidence, segment.BoundingBox); err != nil {
				return err
			}
			fact := facts[i]
			if _, err := tx.Exec(ctx, `INSERT INTO facts (workspace_id,profile_id,id,current_version_id) VALUES ($1,$2,$3,$4)`, workspaceID, profileID, fact.ID, fact.CurrentVersionID); err != nil {
				return err
			}
			version := fact.Versions[0]
			version.ActorID = "parser:plain-lines-v1"
			if err := insertKnowledgeVersion(ctx, tx, workspaceID, profileID, version); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO fact_evidence_links (workspace_id,profile_id,fact_id,fact_version_id,segment_id) VALUES ($1,$2,$3,$4,$5)`, workspaceID, profileID, fact.ID, version.ID, segment.ID); err != nil {
				return err
			}
		}
		if err := appendEvent(ctx, tx, workspaceID, "knowledge.source.created.v1", "profile_source", createdSourceID, map[string]string{"profileId": profileID, "sourceId": createdSourceID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, workspaceID, "knowledge.source.create", "profile_source", source.ID)
	})
	return source, err
}

func insertKnowledgeVersion(ctx context.Context, tx pgx.Tx, workspaceID, profileID string, version knowledge.Version) error {
	_, err := tx.Exec(ctx, `INSERT INTO fact_versions (workspace_id,profile_id,id,fact_id,number,statement,status,sensitive,actor_id,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		workspaceID, profileID, version.ID, version.FactID, version.Number, version.Statement, version.Status, version.Sensitive, version.ActorID, version.CreatedAt)
	return err
}

func (r *KnowledgeRepository) Snapshot(ctx context.Context, workspaceID, profileID string) (knowledge.Snapshot, error) {
	result := knowledge.Snapshot{}
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		if err := knowledgeProfile(ctx, tx, workspaceID, profileID); err != nil {
			return err
		}
		// A single statement gives export and review a consistent MVCC snapshot.
		var data []byte
		err := tx.QueryRow(ctx, `SELECT jsonb_build_object(
			'schemaVersion',1,'profileId',$2::text,
			'sources',COALESCE((SELECT jsonb_agg(jsonb_build_object('id',s.id,'name',s.name,'text',s.text,'hash',s.hash,'mediaType',s.media_type,'parserVersion',s.parser_version,'state',s.state,'createdAt',s.created_at) ORDER BY s.created_at,s.id) FROM profile_sources s WHERE s.workspace_id=$1 AND s.profile_id=$2),'[]'::jsonb),
			'segments',COALESCE((SELECT jsonb_agg(jsonb_strip_nulls(jsonb_build_object('id',s.id,'sourceId',s.source_id,'page',s.page_number,'paragraph',s.paragraph,'text',s.text,'hash',s.hash,'confidence',s.confidence,'boundingBox',s.bounding_box)) ORDER BY s.source_id,s.page_number NULLS FIRST,s.paragraph) FROM source_segments s WHERE s.workspace_id=$1 AND s.profile_id=$2),'[]'::jsonb),
			'facts',COALESCE((SELECT jsonb_agg(jsonb_build_object('id',f.id,'evidenceSegmentIds',COALESCE((SELECT jsonb_agg(e.segment_id ORDER BY e.segment_id) FROM fact_evidence_links e WHERE e.workspace_id=$1 AND e.profile_id=$2 AND e.fact_id=f.id AND e.fact_version_id=f.current_version_id),'[]'::jsonb),'currentVersionId',f.current_version_id,'versions',
				(SELECT jsonb_agg(jsonb_build_object('id',v.id,'factId',v.fact_id,'number',v.number,'statement',v.statement,'status',v.status,'sensitive',v.sensitive,'actorId',v.actor_id,'createdAt',v.created_at) ORDER BY v.number DESC) FROM fact_versions v WHERE v.workspace_id=$1 AND v.profile_id=$2 AND v.fact_id=f.id)) ORDER BY f.id) FROM facts f WHERE f.workspace_id=$1 AND f.profile_id=$2),'[]'::jsonb)
		)`, workspaceID, profileID).Scan(&data)
		if err != nil {
			return err
		}
		return json.Unmarshal(data, &result)
	})
	return result, err
}

func (r *KnowledgeRepository) Review(ctx context.Context, workspaceID, profileID, factID string, input knowledge.Review) (knowledge.Version, error) {
	version := knowledge.Version{ID: id.New("fv"), FactID: factID, Statement: input.Statement, Status: input.Status, Sensitive: input.Sensitive, ActorID: requestcontext.ActorID(ctx), CreatedAt: time.Now().UTC()}
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		var current string
		err := tx.QueryRow(ctx, `SELECT current_version_id FROM facts WHERE workspace_id=$1 AND profile_id=$2 AND id=$3 FOR UPDATE`, workspaceID, profileID, factID).Scan(&current)
		if errors.Is(err, pgx.ErrNoRows) {
			return knowledge.ErrNotFound
		}
		if err != nil {
			return err
		}
		if current != input.ExpectedVersionID {
			return knowledge.ErrConflict
		}
		if err := tx.QueryRow(ctx, `SELECT number+1 FROM fact_versions WHERE workspace_id=$1 AND profile_id=$2 AND fact_id=$3 AND id=$4`, workspaceID, profileID, factID, current).Scan(&version.Number); err != nil {
			return err
		}
		if err := insertKnowledgeVersion(ctx, tx, workspaceID, profileID, version); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO fact_evidence_links (workspace_id,profile_id,fact_id,fact_version_id,segment_id)
			SELECT workspace_id,profile_id,fact_id,$5,segment_id FROM fact_evidence_links
			WHERE workspace_id=$1 AND profile_id=$2 AND fact_id=$3 AND fact_version_id=$4`, workspaceID, profileID, factID, current, version.ID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE facts SET current_version_id=$4 WHERE workspace_id=$1 AND profile_id=$2 AND id=$3`, workspaceID, profileID, factID, version.ID); err != nil {
			return err
		}
		return appendAudit(ctx, tx, workspaceID, "knowledge.fact.review", "fact", factID)
	})
	return version, err
}

func (r *KnowledgeRepository) Search(ctx context.Context, workspaceID, profileID, query string) ([]knowledge.Version, error) {
	result := []knowledge.Version{}
	err := withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		if err := knowledgeProfile(ctx, tx, workspaceID, profileID); err != nil {
			return err
		}
		rows, err := tx.Query(ctx, `SELECT v.id,v.fact_id,v.number,v.statement,v.status,v.sensitive,v.actor_id,v.created_at
			FROM facts f JOIN fact_versions v
			  ON (v.workspace_id,v.profile_id,v.fact_id,v.id)=(f.workspace_id,f.profile_id,f.id,f.current_version_id)
			WHERE f.workspace_id=$1 AND f.profile_id=$2
			  AND v.status='user_confirmed' AND NOT v.sensitive
			  AND to_tsvector('simple',v.statement) @@ websearch_to_tsquery('simple',$3)
			ORDER BY ts_rank_cd(to_tsvector('simple',v.statement),websearch_to_tsquery('simple',$3)) DESC,v.created_at DESC
			LIMIT 50`, workspaceID, profileID, query)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var version knowledge.Version
			if err := rows.Scan(&version.ID, &version.FactID, &version.Number, &version.Statement, &version.Status, &version.Sensitive, &version.ActorID, &version.CreatedAt); err != nil {
				return err
			}
			result = append(result, version)
		}
		return rows.Err()
	})
	return result, err
}

func (r *KnowledgeRepository) DeleteSource(ctx context.Context, workspaceID, profileID, sourceID string) error {
	return withWorkspaceTx(ctx, r.pool, workspaceID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM facts f
			WHERE f.workspace_id=$1 AND f.profile_id=$2
			  AND EXISTS (
			      SELECT 1 FROM fact_evidence_links e JOIN source_segments s
			        ON (s.workspace_id,s.profile_id,s.id)=(e.workspace_id,e.profile_id,e.segment_id)
			      WHERE e.workspace_id=f.workspace_id AND e.profile_id=f.profile_id AND e.fact_id=f.id AND s.source_id=$3
			  )
			  AND NOT EXISTS (
			      SELECT 1 FROM fact_evidence_links e JOIN source_segments s
			        ON (s.workspace_id,s.profile_id,s.id)=(e.workspace_id,e.profile_id,e.segment_id)
			      WHERE e.workspace_id=f.workspace_id AND e.profile_id=f.profile_id AND e.fact_id=f.id AND s.source_id<>$3
			  )`, workspaceID, profileID, sourceID); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, `DELETE FROM profile_sources WHERE workspace_id=$1 AND profile_id=$2 AND id=$3`, workspaceID, profileID, sourceID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return knowledge.ErrNotFound
		}
		// Sources exclusively own their facts in M2a; no source text is copied into events or audit metadata.
		if err := appendEvent(ctx, tx, workspaceID, "knowledge.source.deleted.v1", "profile_source", sourceID, map[string]string{"profileId": profileID, "sourceId": sourceID}); err != nil {
			return err
		}
		return appendAudit(ctx, tx, workspaceID, "knowledge.source.delete", "profile_source", sourceID)
	})
}
