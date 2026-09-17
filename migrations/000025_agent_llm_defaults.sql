CREATE TABLE agent_llm_defaults (
    workspace_id text NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    agent text NOT NULL CHECK (agent IN ('writer', 'template_applier', 'visual_reviewer', 'job_import', 'job_hunter')),
    connection_id text NOT NULL REFERENCES llm_connections(id) ON DELETE CASCADE,
    model text NOT NULL CHECK (model <> '' AND octet_length(model) <= 200),
    updated_at timestamptz NOT NULL,
    PRIMARY KEY (workspace_id, agent)
);

CREATE INDEX agent_llm_defaults_connection_idx ON agent_llm_defaults (connection_id);

ALTER TABLE agent_llm_defaults ENABLE ROW LEVEL SECURITY;
ALTER TABLE agent_llm_defaults FORCE ROW LEVEL SECURITY;

CREATE POLICY agent_llm_defaults_workspace_policy ON agent_llm_defaults
    USING (workspace_id = current_setting('app.workspace_id', true))
    WITH CHECK (workspace_id = current_setting('app.workspace_id', true));
