ALTER TABLE profile_sources DROP CONSTRAINT profile_sources_text_check;
ALTER TABLE profile_sources ADD CONSTRAINT profile_sources_text_check
    CHECK (octet_length(text) BETWEEN 1 AND 10485760);
ALTER TABLE profile_sources DROP CONSTRAINT profile_sources_media_type_check;
ALTER TABLE profile_sources ADD CONSTRAINT profile_sources_media_type_check CHECK (media_type IN (
    'text/plain',
    'application/x-tex',
    'application/msword',
    'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    'application/pdf',
    'image/png',
    'image/jpeg'
));
ALTER TABLE profile_sources ADD CONSTRAINT profile_sources_parser_version_size_check
    CHECK (octet_length(parser_version) BETWEEN 1 AND 100);

CREATE TABLE document_uploads (
    workspace_id text NOT NULL,
    profile_id text NOT NULL,
    id text NOT NULL,
    object_id text NOT NULL,
    name text NOT NULL CHECK (octet_length(name) BETWEEN 1 AND 200),
    declared_media_type text NOT NULL,
    state text NOT NULL DEFAULT 'staged' CHECK (state IN (
        'staged', 'queued', 'ready', 'needs_user_action', 'security_quarantine', 'failed'
    )),
    job_id text,
    source_id text,
    error_code text,
    error_message text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, profile_id, id),
    UNIQUE (workspace_id, object_id),
    FOREIGN KEY (workspace_id, profile_id) REFERENCES profiles(workspace_id, id) ON DELETE CASCADE,
    FOREIGN KEY (job_id) REFERENCES durable_jobs(id) ON DELETE SET NULL,
    FOREIGN KEY (workspace_id, profile_id, source_id) REFERENCES profile_sources(workspace_id, profile_id, id)
        ON DELETE SET NULL (source_id)
);

CREATE INDEX document_uploads_profile_created_idx
    ON document_uploads (workspace_id, profile_id, created_at DESC);

ALTER TABLE document_uploads ENABLE ROW LEVEL SECURITY;
ALTER TABLE document_uploads FORCE ROW LEVEL SECURITY;
CREATE POLICY document_uploads_workspace ON document_uploads
    USING (workspace_id = current_setting('app.workspace_id', true))
    WITH CHECK (workspace_id = current_setting('app.workspace_id', true));

CREATE FUNCTION claim_durable_job_kind(p_worker text, p_lease_seconds integer, p_kind text)
RETURNS SETOF durable_jobs
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    UPDATE durable_jobs
    SET state = 'failed', lease_owner = NULL, lease_expires_at = NULL,
        error_class = CASE WHEN deadline_at IS NOT NULL AND deadline_at <= now() THEN 'deadline_exceeded' ELSE 'attempts_exhausted' END,
        error_message = CASE WHEN deadline_at IS NOT NULL AND deadline_at <= now() THEN 'The job deadline expired before completion.' ELSE 'The worker lease expired after the maximum number of attempts.' END,
        updated_at = now()
    WHERE kind = p_kind AND (
        (state IN ('queued', 'retry_wait') AND deadline_at IS NOT NULL AND deadline_at <= now())
        OR (state = 'running' AND lease_expires_at <= now() AND (attempt >= max_attempts OR (deadline_at IS NOT NULL AND deadline_at <= now())))
    );

    RETURN QUERY
    UPDATE durable_jobs
    SET state = 'running', attempt = attempt + 1, lease_owner = p_worker,
        lease_expires_at = now() + make_interval(secs => p_lease_seconds), heartbeat_at = now(), updated_at = now()
    WHERE id = (
        SELECT id FROM durable_jobs
        WHERE kind = p_kind AND attempt < max_attempts AND (deadline_at IS NULL OR deadline_at > now())
          AND ((state IN ('queued', 'retry_wait') AND available_at <= now()) OR (state = 'running' AND lease_expires_at <= now()))
        ORDER BY available_at, created_at FOR UPDATE SKIP LOCKED LIMIT 1
    )
    RETURNING *;
END;
$$;
