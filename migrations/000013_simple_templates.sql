CREATE TABLE templates (
    id text PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (name <> '' AND octet_length(name) <= 120),
    kind text NOT NULL CHECK (kind IN ('resume', 'cover_letter')),
    format text NOT NULL CHECK (format IN ('latex', 'doc', 'docx')),
    description text NOT NULL DEFAULT '' CHECK (octet_length(description) <= 1000),
    source_name text NOT NULL CHECK (source_name <> '' AND octet_length(source_name) <= 200),
    declared_media_type text NOT NULL CHECK (octet_length(declared_media_type) <= 200),
    object_id text NOT NULL,
    content text,
    state text NOT NULL DEFAULT 'staged' CHECK (state IN (
        'staged', 'queued', 'ready', 'needs_user_action', 'security_quarantine', 'failed'
    )),
    job_id text REFERENCES durable_jobs(id) ON DELETE SET NULL,
    error_code text,
    error_message text,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (workspace_id, object_id)
);

CREATE INDEX templates_workspace_created_idx ON templates (workspace_id, created_at DESC);

ALTER TABLE templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE templates FORCE ROW LEVEL SECURITY;
CREATE POLICY templates_workspace_policy ON templates
    USING (workspace_id = current_setting('app.workspace_id', true))
    WITH CHECK (workspace_id = current_setting('app.workspace_id', true));
