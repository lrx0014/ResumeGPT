ALTER TABLE profiles ADD CONSTRAINT profiles_workspace_id_unique UNIQUE (workspace_id, id);

CREATE TABLE profile_sources (
    workspace_id text NOT NULL,
    profile_id text NOT NULL,
    id text NOT NULL,
    name text NOT NULL,
    text text NOT NULL CHECK (octet_length(text) BETWEEN 1 AND 65536),
    hash text NOT NULL,
    media_type text NOT NULL DEFAULT 'text/plain' CHECK (media_type = 'text/plain'),
    parser_version text NOT NULL DEFAULT 'plain-lines-v1',
    state text NOT NULL DEFAULT 'ready' CHECK (state = 'ready'),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, profile_id, id),
    UNIQUE (workspace_id, profile_id, hash),
    FOREIGN KEY (workspace_id, profile_id) REFERENCES profiles(workspace_id, id) ON DELETE CASCADE
);

CREATE TABLE source_segments (
    workspace_id text NOT NULL,
    profile_id text NOT NULL,
    id text NOT NULL,
    source_id text NOT NULL,
    paragraph integer NOT NULL CHECK (paragraph > 0),
    text text NOT NULL,
    hash text NOT NULL,
    PRIMARY KEY (workspace_id, profile_id, id),
    FOREIGN KEY (workspace_id, profile_id, source_id) REFERENCES profile_sources ON DELETE CASCADE
);

CREATE TABLE facts (
    workspace_id text NOT NULL,
    profile_id text NOT NULL,
    id text NOT NULL,
    segment_id text NOT NULL,
    current_version_id text NOT NULL,
    PRIMARY KEY (workspace_id, profile_id, id),
    FOREIGN KEY (workspace_id, profile_id, segment_id) REFERENCES source_segments ON DELETE CASCADE
);

CREATE TABLE fact_versions (
    workspace_id text NOT NULL,
    profile_id text NOT NULL,
    id text NOT NULL,
    fact_id text NOT NULL,
    number integer NOT NULL CHECK (number > 0),
    fact_type text NOT NULL DEFAULT 'statement' CHECK (fact_type = 'statement'),
    schema_version integer NOT NULL DEFAULT 1 CHECK (schema_version = 1),
    statement text NOT NULL CHECK (octet_length(statement) BETWEEN 1 AND 4000),
    status text NOT NULL CHECK (status IN ('extracted', 'user_asserted', 'user_confirmed', 'disputed', 'rejected')),
    sensitive boolean NOT NULL DEFAULT true,
    actor_id text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, profile_id, fact_id, id),
    UNIQUE (workspace_id, profile_id, fact_id, number),
    FOREIGN KEY (workspace_id, profile_id, fact_id) REFERENCES facts ON DELETE CASCADE
);

ALTER TABLE facts ADD CONSTRAINT facts_current_version_fk
    FOREIGN KEY (workspace_id, profile_id, id, current_version_id)
    REFERENCES fact_versions(workspace_id, profile_id, fact_id, id)
    DEFERRABLE INITIALLY DEFERRED;

CREATE FUNCTION prevent_knowledge_update() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'Source material and fact versions are immutable; append a new version instead';
END;
$$;
CREATE TRIGGER immutable_sources BEFORE UPDATE ON profile_sources FOR EACH ROW EXECUTE FUNCTION prevent_knowledge_update();
CREATE TRIGGER immutable_segments BEFORE UPDATE ON source_segments FOR EACH ROW EXECUTE FUNCTION prevent_knowledge_update();
CREATE TRIGGER immutable_fact_versions BEFORE UPDATE ON fact_versions FOR EACH ROW EXECUTE FUNCTION prevent_knowledge_update();

ALTER TABLE profile_sources ENABLE ROW LEVEL SECURITY;
ALTER TABLE profile_sources FORCE ROW LEVEL SECURITY;
ALTER TABLE source_segments ENABLE ROW LEVEL SECURITY;
ALTER TABLE source_segments FORCE ROW LEVEL SECURITY;
ALTER TABLE facts ENABLE ROW LEVEL SECURITY;
ALTER TABLE facts FORCE ROW LEVEL SECURITY;
ALTER TABLE fact_versions ENABLE ROW LEVEL SECURITY;
ALTER TABLE fact_versions FORCE ROW LEVEL SECURITY;

CREATE POLICY sources_workspace ON profile_sources USING (workspace_id = current_setting('app.workspace_id', true)) WITH CHECK (workspace_id = current_setting('app.workspace_id', true));
CREATE POLICY segments_workspace ON source_segments USING (workspace_id = current_setting('app.workspace_id', true)) WITH CHECK (workspace_id = current_setting('app.workspace_id', true));
CREATE POLICY facts_workspace ON facts USING (workspace_id = current_setting('app.workspace_id', true)) WITH CHECK (workspace_id = current_setting('app.workspace_id', true));
CREATE POLICY versions_workspace ON fact_versions USING (workspace_id = current_setting('app.workspace_id', true)) WITH CHECK (workspace_id = current_setting('app.workspace_id', true));
