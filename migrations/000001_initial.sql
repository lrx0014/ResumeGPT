CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS schema_migrations (
    version text PRIMARY KEY,
    applied_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE workspaces (
    id text PRIMARY KEY,
    name text NOT NULL,
    kind text NOT NULL CHECK (kind IN ('personal', 'team')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id text PRIMARY KEY,
    issuer text NOT NULL,
    subject text NOT NULL,
    email text,
    display_name text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (issuer, subject)
);

CREATE TABLE workspace_memberships (
    workspace_id text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id text NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role text NOT NULL CHECK (role IN ('owner', 'admin', 'editor', 'viewer')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, user_id)
);

CREATE TABLE profiles (
    id text PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name text NOT NULL,
    domain text NOT NULL DEFAULT '',
    default_language text NOT NULL DEFAULT 'en-US',
    description text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (workspace_id, name)
);

CREATE INDEX profiles_workspace_created_idx ON profiles (workspace_id, created_at DESC);

CREATE TABLE jobs (
    id text PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    title text NOT NULL,
    company text NOT NULL,
    location text NOT NULL DEFAULT '',
    source_url text NOT NULL DEFAULT '',
    description text NOT NULL DEFAULT '',
    status text NOT NULL DEFAULT 'interested'
        CHECK (status IN ('interested', 'preparing', 'applied', 'screening', 'interview', 'offer', 'accepted', 'rejected', 'withdrawn')),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);

CREATE INDEX jobs_workspace_created_idx ON jobs (workspace_id, created_at DESC);

CREATE TABLE durable_jobs (
    id text PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    kind text NOT NULL,
    state text NOT NULL DEFAULT 'queued'
        CHECK (state IN ('queued', 'running', 'retry_wait', 'succeeded', 'failed', 'cancelled')),
    idempotency_key text NOT NULL,
    payload jsonb NOT NULL DEFAULT '{}'::jsonb,
    attempt integer NOT NULL DEFAULT 0 CHECK (attempt >= 0),
    max_attempts integer NOT NULL DEFAULT 5 CHECK (max_attempts > 0),
    available_at timestamptz NOT NULL DEFAULT now(),
    deadline_at timestamptz,
    lease_owner text,
    lease_expires_at timestamptz,
    heartbeat_at timestamptz,
    error_class text,
    error_message text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (workspace_id, kind, idempotency_key)
);

CREATE INDEX durable_jobs_claim_idx ON durable_jobs (available_at, created_at)
    WHERE state IN ('queued', 'retry_wait');

CREATE TABLE outbox_events (
    id text PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    topic text NOT NULL,
    aggregate_type text NOT NULL,
    aggregate_id text NOT NULL,
    payload jsonb NOT NULL,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz,
    attempt integer NOT NULL DEFAULT 0,
    last_error text
);

CREATE INDEX outbox_unpublished_idx ON outbox_events (occurred_at)
    WHERE published_at IS NULL;

CREATE TABLE inbox_messages (
    consumer text NOT NULL,
    message_id text NOT NULL,
    processed_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (consumer, message_id)
);

CREATE TABLE audit_events (
    id text PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    actor_id text NOT NULL,
    action text NOT NULL,
    resource_type text NOT NULL,
    resource_id text NOT NULL,
    result text NOT NULL,
    request_id text,
    metadata jsonb NOT NULL DEFAULT '{}'::jsonb,
    occurred_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX audit_workspace_occurred_idx ON audit_events (workspace_id, occurred_at DESC);

ALTER TABLE profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE profiles FORCE ROW LEVEL SECURITY;
ALTER TABLE jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE jobs FORCE ROW LEVEL SECURITY;
ALTER TABLE durable_jobs ENABLE ROW LEVEL SECURITY;
ALTER TABLE durable_jobs FORCE ROW LEVEL SECURITY;
ALTER TABLE outbox_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE outbox_events FORCE ROW LEVEL SECURITY;
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events FORCE ROW LEVEL SECURITY;

CREATE POLICY profiles_workspace_policy ON profiles
    USING (workspace_id = current_setting('app.workspace_id', true))
    WITH CHECK (workspace_id = current_setting('app.workspace_id', true));
CREATE POLICY jobs_workspace_policy ON jobs
    USING (workspace_id = current_setting('app.workspace_id', true))
    WITH CHECK (workspace_id = current_setting('app.workspace_id', true));
CREATE POLICY durable_jobs_workspace_policy ON durable_jobs
    USING (workspace_id = current_setting('app.workspace_id', true))
    WITH CHECK (workspace_id = current_setting('app.workspace_id', true));
CREATE POLICY outbox_events_workspace_policy ON outbox_events
    USING (workspace_id = current_setting('app.workspace_id', true))
    WITH CHECK (workspace_id = current_setting('app.workspace_id', true));
CREATE POLICY audit_events_workspace_policy ON audit_events
    USING (workspace_id = current_setting('app.workspace_id', true))
    WITH CHECK (workspace_id = current_setting('app.workspace_id', true));

