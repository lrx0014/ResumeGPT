CREATE TABLE fact_evidence_links (
    workspace_id text NOT NULL,
    profile_id text NOT NULL,
    fact_id text NOT NULL,
    fact_version_id text NOT NULL,
    segment_id text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, profile_id, fact_id, fact_version_id, segment_id),
    FOREIGN KEY (workspace_id, profile_id, fact_id, fact_version_id)
        REFERENCES fact_versions(workspace_id, profile_id, fact_id, id) ON DELETE CASCADE,
    FOREIGN KEY (workspace_id, profile_id, segment_id)
        REFERENCES source_segments(workspace_id, profile_id, id) ON DELETE CASCADE
);

INSERT INTO fact_evidence_links (workspace_id, profile_id, fact_id, fact_version_id, segment_id)
SELECT workspace_id, profile_id, id, current_version_id, segment_id FROM facts;

ALTER TABLE facts DROP CONSTRAINT facts_workspace_id_profile_id_segment_id_fkey;
ALTER TABLE facts DROP COLUMN segment_id;

ALTER TABLE fact_evidence_links ENABLE ROW LEVEL SECURITY;
ALTER TABLE fact_evidence_links FORCE ROW LEVEL SECURITY;
CREATE POLICY evidence_links_workspace ON fact_evidence_links
    USING (workspace_id = current_setting('app.workspace_id', true))
    WITH CHECK (workspace_id = current_setting('app.workspace_id', true));
