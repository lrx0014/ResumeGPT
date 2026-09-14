CREATE TABLE workspace_settings (
    workspace_id text PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    interface_language text NOT NULL DEFAULT 'en' CHECK (interface_language IN ('en', 'de')),
    theme text NOT NULL DEFAULT 'system' CHECK (theme IN ('system', 'light', 'dark')),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE llm_connections (
    id text PRIMARY KEY,
    workspace_id text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name text NOT NULL CHECK (name <> '' AND octet_length(name) <= 120),
    execution_mode text NOT NULL CHECK (execution_mode IN ('cloud', 'local')),
    provider text NOT NULL CHECK (provider IN ('openai', 'openai_compatible', 'ollama')),
    base_url text NOT NULL CHECK (base_url <> '' AND octet_length(base_url) <= 2048),
    api_token_ciphertext bytea,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    UNIQUE (workspace_id, name)
);

CREATE INDEX llm_connections_workspace_created_idx ON llm_connections (workspace_id, created_at);

ALTER TABLE workspace_settings ENABLE ROW LEVEL SECURITY;
ALTER TABLE workspace_settings FORCE ROW LEVEL SECURITY;
ALTER TABLE llm_connections ENABLE ROW LEVEL SECURITY;
ALTER TABLE llm_connections FORCE ROW LEVEL SECURITY;

CREATE POLICY workspace_settings_workspace_policy ON workspace_settings
    USING (workspace_id = current_setting('app.workspace_id', true))
    WITH CHECK (workspace_id = current_setting('app.workspace_id', true));

CREATE POLICY llm_connections_workspace_policy ON llm_connections
    USING (workspace_id = current_setting('app.workspace_id', true))
    WITH CHECK (workspace_id = current_setting('app.workspace_id', true));
