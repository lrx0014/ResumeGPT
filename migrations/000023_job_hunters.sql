CREATE TABLE job_hunters (
    id text PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (name <> '' AND octet_length(name) <= 120),
    role_query text NOT NULL CHECK (role_query <> '' AND octet_length(role_query) <= 300),
    location text NOT NULL DEFAULT '' CHECK (octet_length(location) <= 300),
    work_mode text NOT NULL DEFAULT '' CHECK (octet_length(work_mode) <= 100),
    employment_type text NOT NULL DEFAULT '' CHECK (octet_length(employment_type) <= 100),
    experience_years integer CHECK (experience_years IS NULL OR experience_years BETWEEN 0 AND 60),
    keywords text NOT NULL DEFAULT '' CHECK (octet_length(keywords) <= 1000),
    additional_prompt text NOT NULL DEFAULT '' CHECK (octet_length(additional_prompt) <= 4000),
    connection_id text NOT NULL,
    model text NOT NULL CHECK (model <> '' AND octet_length(model) <= 200),
    interval_minutes integer NOT NULL CHECK (interval_minutes IN (360, 720, 1440, 10080)),
    enabled boolean NOT NULL DEFAULT true,
    next_run_at timestamptz NOT NULL,
    last_run_at timestamptz,
    last_state text NOT NULL DEFAULT 'never' CHECK (last_state IN ('never', 'queued', 'running', 'succeeded', 'failed')),
    last_error text NOT NULL DEFAULT '' CHECK (octet_length(last_error) <= 2000),
    last_found_count integer NOT NULL DEFAULT 0 CHECK (last_found_count >= 0),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE INDEX job_hunters_workspace_created_idx ON job_hunters (workspace_id, created_at DESC);
CREATE INDEX job_hunters_due_idx ON job_hunters (next_run_at) WHERE enabled;

ALTER TABLE job_hunters ENABLE ROW LEVEL SECURITY;
ALTER TABLE job_hunters FORCE ROW LEVEL SECURITY;

CREATE POLICY job_hunters_workspace_policy ON job_hunters
    USING (workspace_id = current_setting('app.workspace_id', true))
    WITH CHECK (workspace_id = current_setting('app.workspace_id', true));

ALTER TABLE jobs
    ADD COLUMN origin text NOT NULL DEFAULT 'manual' CHECK (origin IN ('manual', 'url_import', 'hunter')),
    ADD COLUMN hunter_id text REFERENCES job_hunters(id) ON DELETE SET NULL;

UPDATE jobs SET origin = 'url_import' WHERE import_state <> 'manual';

CREATE INDEX jobs_workspace_origin_idx ON jobs (workspace_id, origin, created_at DESC);
CREATE INDEX jobs_hunter_idx ON jobs (workspace_id, hunter_id) WHERE hunter_id IS NOT NULL;

CREATE OR REPLACE FUNCTION schedule_due_job_hunts(batch_size integer DEFAULT 20)
RETURNS integer
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    candidate record;
    task_id text;
    scheduled_count integer := 0;
BEGIN
    FOR candidate IN
        SELECT id, workspace_id, interval_minutes, next_run_at
        FROM job_hunters
        WHERE enabled AND next_run_at <= now()
          AND NOT EXISTS (
              SELECT 1 FROM durable_jobs
              WHERE workspace_id = job_hunters.workspace_id
                AND kind = 'job.hunt.v1'
                AND payload->>'hunterId' = job_hunters.id
                AND state IN ('queued', 'running', 'retry_wait')
          )
        ORDER BY next_run_at, id
        FOR UPDATE SKIP LOCKED
        LIMIT LEAST(GREATEST(batch_size, 1), 100)
    LOOP
        task_id := 'task_' || substr(md5(candidate.id || clock_timestamp()::text || random()::text), 1, 24);
        PERFORM set_config('app.workspace_id', candidate.workspace_id, true);
        INSERT INTO durable_jobs (id, workspace_id, kind, idempotency_key, payload, max_attempts, available_at)
        VALUES (
            task_id,
            candidate.workspace_id,
            'job.hunt.v1',
            candidate.id || ':' || extract(epoch FROM candidate.next_run_at)::bigint::text,
            jsonb_build_object('hunterId', candidate.id),
            3,
            now()
        ) ON CONFLICT (workspace_id, kind, idempotency_key) DO NOTHING;

        UPDATE job_hunters
        SET next_run_at = now() + make_interval(mins => interval_minutes),
            last_state = 'queued', last_error = '', updated_at = now()
        WHERE id = candidate.id;
        scheduled_count := scheduled_count + 1;
    END LOOP;
    RETURN scheduled_count;
END;
$$;
