CREATE TABLE job_hunter_review_items (
    id text PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    hunter_id text NOT NULL REFERENCES job_hunters(id) ON DELETE CASCADE,
    source_url text NOT NULL CHECK (source_url <> '' AND octet_length(source_url) <= 2048),
    failure_code text NOT NULL DEFAULT '' CHECK (octet_length(failure_code) <= 120),
    failure_message text NOT NULL DEFAULT '' CHECK (octet_length(failure_message) <= 2000),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (workspace_id, hunter_id, source_url)
);

CREATE INDEX job_hunter_review_items_hunter_idx
    ON job_hunter_review_items (workspace_id, hunter_id, created_at DESC);

INSERT INTO job_hunter_review_items
    (id, workspace_id, hunter_id, source_url, failure_code, failure_message, created_at, updated_at)
SELECT
    'review_' || substr(md5(workspace_id || '|' || id), 1, 24),
    workspace_id,
    hunter_id,
    source_url,
    'legacy_import_failed',
    import_error,
    created_at,
    updated_at
FROM jobs
WHERE origin = 'hunter'
  AND hunter_id IS NOT NULL
  AND source_url <> ''
  AND import_state IN ('needs_user_action', 'failed')
ON CONFLICT (workspace_id, hunter_id, source_url) DO NOTHING;

DELETE FROM jobs
WHERE origin = 'hunter'
  AND hunter_id IS NOT NULL
  AND import_state IN ('needs_user_action', 'failed');

ALTER TABLE job_hunter_review_items ENABLE ROW LEVEL SECURITY;
ALTER TABLE job_hunter_review_items FORCE ROW LEVEL SECURITY;

CREATE POLICY job_hunter_review_items_workspace_policy ON job_hunter_review_items
    USING (workspace_id = current_setting('app.workspace_id', true))
    WITH CHECK (workspace_id = current_setting('app.workspace_id', true));
