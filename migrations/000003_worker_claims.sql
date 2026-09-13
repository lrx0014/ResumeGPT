ALTER TABLE outbox_events
    ADD COLUMN locked_by text,
    ADD COLUMN locked_until timestamptz;

CREATE OR REPLACE FUNCTION claim_durable_job(p_worker text, p_lease_seconds integer)
RETURNS SETOF durable_jobs
LANGUAGE sql
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
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
        WHERE state IN ('queued', 'retry_wait')
          AND available_at <= now()
          AND (deadline_at IS NULL OR deadline_at > now())
        ORDER BY available_at, created_at
        FOR UPDATE SKIP LOCKED
        LIMIT 1
    )
    RETURNING *;
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
          AND (locked_until IS NULL OR locked_until < now())
        ORDER BY occurred_at
        FOR UPDATE SKIP LOCKED
        LIMIT 1
    )
    RETURNING *;
$$;

