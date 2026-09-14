ALTER TABLE profiles
    ADD COLUMN target_role text NOT NULL DEFAULT '',
    ADD COLUMN content text NOT NULL DEFAULT '' CHECK (octet_length(content) <= 1048576),
    ADD COLUMN avatar_object_id text;

UPDATE profiles p
SET target_role = p.domain,
    content = concat_ws(E'\n\n',
        NULLIF(p.description, ''),
        (
            SELECT string_agg(concat('## ', s.name, E'\n\n', s.text), E'\n\n' ORDER BY s.created_at, s.id)
            FROM profile_sources s
            WHERE s.workspace_id = p.workspace_id AND s.profile_id = p.id
        )
    );

ALTER TABLE profiles
    DROP COLUMN domain,
    DROP COLUMN description;

ALTER TABLE document_uploads
    DROP CONSTRAINT document_uploads_workspace_id_profile_id_source_id_fkey,
    DROP COLUMN source_id,
    ADD COLUMN extracted_text text CHECK (extracted_text IS NULL OR octet_length(extracted_text) <= 1048576);

DROP TABLE fact_evidence_links;
DROP TABLE fact_versions CASCADE;
DROP TABLE facts;
DROP TABLE source_segments;
DROP TABLE profile_sources;
DROP FUNCTION prevent_knowledge_update();
