ALTER TABLE job_hunters
    ADD COLUMN profile_id text REFERENCES profiles(id) ON DELETE SET NULL,
    ADD COLUMN max_results integer NOT NULL DEFAULT 10 CHECK (max_results BETWEEN 1 AND 10);

CREATE INDEX job_hunters_profile_idx
    ON job_hunters (workspace_id, profile_id)
    WHERE profile_id IS NOT NULL;
