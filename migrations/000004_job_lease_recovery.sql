CREATE INDEX durable_jobs_expired_lease_idx ON durable_jobs (lease_expires_at)
    WHERE state = 'running';

CREATE OR REPLACE FUNCTION claim_durable_job(p_worker text, p_lease_seconds integer)
RETURNS SETOF durable_jobs
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
BEGIN
    UPDATE durable_jobs
    SET state = 'failed',
        lease_owner = NULL,
        lease_expires_at = NULL,
        error_class = CASE
            WHEN deadline_at IS NOT NULL AND deadline_at <= now() THEN 'deadline_exceeded'
            ELSE 'attempts_exhausted'
        END,
        error_message = CASE
            WHEN deadline_at IS NOT NULL AND deadline_at <= now() THEN 'The job deadline expired before completion.'
            ELSE 'The worker lease expired after the maximum number of attempts.'
        END,
        updated_at = now()
    WHERE (state IN ('queued', 'retry_wait') AND deadline_at IS NOT NULL AND deadline_at <= now())
       OR (state = 'running' AND lease_expires_at <= now() AND (attempt >= max_attempts OR (deadline_at IS NOT NULL AND deadline_at <= now())));

    RETURN QUERY
    UPDATE durable_jobs
    SET state = 'running',
        attempt = attempt + 1,
        lease_owner = p_worker,
        lease_expires_at = now() + make_interval(secs => p_lease_seconds),
        heartbeat_at = now(),
        updated_at = now()
    WHERE id = (
        SELECT id
        FROM durable_jobs
        WHERE attempt < max_attempts
          AND (deadline_at IS NULL OR deadline_at > now())
          AND (
              (state IN ('queued', 'retry_wait') AND available_at <= now())
              OR (state = 'running' AND lease_expires_at <= now())
          )
        ORDER BY available_at, created_at
        FOR UPDATE SKIP LOCKED
        LIMIT 1
    )
    RETURNING *;
END;
$$;

CREATE OR REPLACE FUNCTION claim_outbox_event(p_worker text, p_lease_seconds integer)
RETURNS SETOF outbox_events
LANGUAGE sql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
    UPDATE outbox_events
    SET locked_by = p_worker,
        locked_until = now() + make_interval(secs => p_lease_seconds),
        attempt = attempt + 1
    WHERE id = (
        SELECT id
        FROM outbox_events
        WHERE published_at IS NULL
          AND (locked_until IS NULL OR locked_until <= now())
        ORDER BY occurred_at
        FOR UPDATE SKIP LOCKED
        LIMIT 1
    )
    RETURNING *;
$$;
