CREATE TABLE durable_job_events (
    id bigserial PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    job_id text NOT NULL REFERENCES durable_jobs(id) ON DELETE CASCADE,
    event_type text NOT NULL CHECK (event_type IN (
        'queued', 'started', 'retry_scheduled', 'completed', 'failed', 'cancelled'
    )),
    state text NOT NULL,
    attempt integer NOT NULL CHECK (attempt >= 0),
    error_class text,
    message text NOT NULL DEFAULT '' CHECK (octet_length(message) <= 4000),
    occurred_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX durable_job_events_job_idx
    ON durable_job_events (workspace_id, job_id, occurred_at DESC, id DESC);

CREATE OR REPLACE FUNCTION record_durable_job_event()
RETURNS trigger
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
DECLARE
    event_name text;
    event_message text;
BEGIN
    IF TG_OP = 'UPDATE'
       AND NEW.state IS NOT DISTINCT FROM OLD.state
       AND NEW.attempt IS NOT DISTINCT FROM OLD.attempt
       AND NEW.error_class IS NOT DISTINCT FROM OLD.error_class
       AND NEW.error_message IS NOT DISTINCT FROM OLD.error_message THEN
        RETURN NEW;
    END IF;

    event_name := CASE NEW.state
        WHEN 'queued' THEN 'queued'
        WHEN 'running' THEN 'started'
        WHEN 'retry_wait' THEN 'retry_scheduled'
        WHEN 'succeeded' THEN 'completed'
        WHEN 'failed' THEN 'failed'
        WHEN 'cancelled' THEN 'cancelled'
    END;
    event_message := CASE NEW.state
        WHEN 'queued' THEN 'Task queued.'
        WHEN 'running' THEN 'Worker started processing the task.'
        WHEN 'retry_wait' THEN 'Task scheduled for another attempt.'
        WHEN 'succeeded' THEN 'Task completed successfully.'
        WHEN 'failed' THEN COALESCE(NULLIF(NEW.error_message, ''), 'Task failed.')
        WHEN 'cancelled' THEN COALESCE(NULLIF(NEW.error_message, ''), 'Task cancelled.')
    END;

    -- Queue workers update durable jobs outside workspace-scoped API
    -- transactions. Scope the trigger insert to the job's own workspace so
    -- event persistence continues to obey row-level security.
    PERFORM set_config('app.workspace_id', NEW.workspace_id, true);

    INSERT INTO durable_job_events (
        workspace_id, job_id, event_type, state, attempt, error_class, message, occurred_at
    ) VALUES (
        NEW.workspace_id, NEW.id, event_name, NEW.state, NEW.attempt,
        NULLIF(NEW.error_class, ''), event_message, now()
    );
    RETURN NEW;
END;
$$;

CREATE TRIGGER durable_job_event_trigger
AFTER INSERT OR UPDATE ON durable_jobs
FOR EACH ROW EXECUTE FUNCTION record_durable_job_event();

INSERT INTO durable_job_events (
    workspace_id, job_id, event_type, state, attempt, error_class, message, occurred_at
)
SELECT workspace_id, id,
    CASE state
        WHEN 'queued' THEN 'queued'
        WHEN 'running' THEN 'started'
        WHEN 'retry_wait' THEN 'retry_scheduled'
        WHEN 'succeeded' THEN 'completed'
        WHEN 'failed' THEN 'failed'
        WHEN 'cancelled' THEN 'cancelled'
    END,
    state, attempt, NULLIF(error_class, ''),
    CASE state
        WHEN 'queued' THEN 'Current state when Task Monitor was enabled.'
        WHEN 'running' THEN 'Task was already running when Task Monitor was enabled.'
        WHEN 'retry_wait' THEN COALESCE(NULLIF(error_message, ''), 'Task was waiting to retry when Task Monitor was enabled.')
        WHEN 'succeeded' THEN 'Task completed before Task Monitor was enabled.'
        WHEN 'failed' THEN COALESCE(NULLIF(error_message, ''), 'Task failed before Task Monitor was enabled.')
        WHEN 'cancelled' THEN COALESCE(NULLIF(error_message, ''), 'Task was cancelled before Task Monitor was enabled.')
    END,
    updated_at
FROM durable_jobs;

ALTER TABLE durable_job_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE durable_job_events FORCE ROW LEVEL SECURITY;

CREATE POLICY durable_job_events_workspace_policy ON durable_job_events
    USING (workspace_id = current_setting('app.workspace_id', true))
    WITH CHECK (workspace_id = current_setting('app.workspace_id', true));
