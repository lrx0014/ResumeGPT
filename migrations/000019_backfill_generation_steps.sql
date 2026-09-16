INSERT INTO generation_steps (id, workspace_id, generation_id, kind, sequence, content, repair_count, created_at)
SELECT 'step_backfill_writer_' || id, workspace_id, id, 'writer_draft', 1, draft, 0, created_at
FROM generation_runs
WHERE COALESCE(draft, '') <> ''
ON CONFLICT DO NOTHING;

INSERT INTO generation_steps (id, workspace_id, generation_id, kind, sequence, content, artifact_object_id, repair_count, created_at)
SELECT 'step_backfill_pdf_' || id, workspace_id, id, 'rendered_pdf',
       CASE WHEN COALESCE(draft, '') <> '' THEN 2 ELSE 1 END,
       COALESCE(rendered_source, ''), artifact_object_id, repair_count, updated_at
FROM generation_runs
WHERE COALESCE(artifact_object_id, '') <> ''
ON CONFLICT DO NOTHING;

INSERT INTO generation_steps (id, workspace_id, generation_id, kind, sequence, feedback, repair_count, created_at)
SELECT 'step_backfill_review_' || id, workspace_id, id,
       CASE WHEN review LIKE 'Template fallback used:%' OR review LIKE 'Visual QA skipped:%' THEN 'system_warning' ELSE 'reviewer_feedback' END,
       CASE
           WHEN COALESCE(draft, '') <> '' AND COALESCE(artifact_object_id, '') <> '' THEN 3
           WHEN COALESCE(draft, '') <> '' OR COALESCE(artifact_object_id, '') <> '' THEN 2
           ELSE 1
       END,
       review, repair_count, updated_at
FROM generation_runs
WHERE COALESCE(review, '') <> ''
ON CONFLICT DO NOTHING;
