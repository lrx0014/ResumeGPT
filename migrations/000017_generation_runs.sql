CREATE TABLE generation_runs (
    id text PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    profile_id text NOT NULL,
    opportunity_id text NOT NULL,
    template_id text NOT NULL,
    document_type text NOT NULL CHECK (document_type IN ('resume', 'cover_letter')),
    language text NOT NULL CHECK (language <> '' AND octet_length(language) <= 40),
    page_target text NOT NULL CHECK (page_target IN ('one_page', 'two_pages', 'flexible')),
    custom_instructions text NOT NULL DEFAULT '' CHECK (octet_length(custom_instructions) <= 4000),
    pipeline_mode text NOT NULL CHECK (pipeline_mode IN ('single', 'multi')),
    writer jsonb NOT NULL,
    renderer jsonb NOT NULL,
    reviewer jsonb NOT NULL,
    profile_snapshot jsonb NOT NULL,
    opportunity_snapshot jsonb NOT NULL,
    template_snapshot jsonb NOT NULL,
    state text NOT NULL CHECK (state IN ('queued', 'running', 'ready', 'failed')),
    stage text NOT NULL,
    draft text,
    rendered_source text,
    review text,
    repair_count integer NOT NULL DEFAULT 0 CHECK (repair_count BETWEEN 0 AND 2),
    artifact_object_id text,
    job_id text NOT NULL REFERENCES durable_jobs(id) ON DELETE RESTRICT,
    error_code text,
    error_message text,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (workspace_id, artifact_object_id)
);

CREATE INDEX generation_runs_workspace_created_idx ON generation_runs (workspace_id, created_at DESC);
ALTER TABLE generation_runs ENABLE ROW LEVEL SECURITY;
ALTER TABLE generation_runs FORCE ROW LEVEL SECURITY;
CREATE POLICY generation_runs_workspace_policy ON generation_runs
    USING (workspace_id = current_setting('app.workspace_id', true))
    WITH CHECK (workspace_id = current_setting('app.workspace_id', true));

