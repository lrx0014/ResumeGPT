ALTER TABLE jobs
    DROP CONSTRAINT jobs_import_state_check;

ALTER TABLE jobs
    ADD CONSTRAINT jobs_import_state_check CHECK (import_state IN (
        'manual', 'queued', 'fetching', 'analyzing', 'ready', 'needs_user_action', 'failed'
    ));
