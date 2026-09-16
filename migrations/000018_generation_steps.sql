CREATE TABLE generation_steps (
    id text PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    generation_id text NOT NULL REFERENCES generation_runs(id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('writer_draft', 'rendered_pdf', 'reviewer_feedback', 'user_prompt', 'system_warning')),
    sequence integer NOT NULL CHECK (sequence > 0),
    content text NOT NULL DEFAULT '',
    feedback text NOT NULL DEFAULT '',
    artifact_object_id text,
    repair_count integer NOT NULL DEFAULT 0 CHECK (repair_count BETWEEN 0 AND 2),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, generation_id, sequence)
);

CREATE INDEX generation_steps_run_sequence_idx ON generation_steps (workspace_id, generation_id, sequence);
ALTER TABLE generation_steps ENABLE ROW LEVEL SECURITY;
ALTER TABLE generation_steps FORCE ROW LEVEL SECURITY;
CREATE POLICY generation_steps_workspace_policy ON generation_steps
    USING (workspace_id = current_setting('app.workspace_id', true))
    WITH CHECK (workspace_id = current_setting('app.workspace_id', true));
