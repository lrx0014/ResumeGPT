ALTER TABLE jobs
    ADD COLUMN country text NOT NULL DEFAULT '' CHECK (octet_length(country) <= 100),
    ADD COLUMN city text NOT NULL DEFAULT '' CHECK (octet_length(city) <= 150),
    ADD COLUMN work_mode text NOT NULL DEFAULT '' CHECK (octet_length(work_mode) <= 100),
    ADD COLUMN employment_type text NOT NULL DEFAULT '' CHECK (octet_length(employment_type) <= 100),
    ADD COLUMN import_state text NOT NULL DEFAULT 'manual' CHECK (import_state IN (
        'manual', 'queued', 'fetching', 'ready', 'needs_user_action', 'failed'
    )),
    ADD COLUMN import_error text NOT NULL DEFAULT '' CHECK (octet_length(import_error) <= 2000);

ALTER TABLE jobs
    ADD CONSTRAINT jobs_title_size_check CHECK (octet_length(title) <= 300),
    ADD CONSTRAINT jobs_company_size_check CHECK (octet_length(company) <= 300),
    ADD CONSTRAINT jobs_location_size_check CHECK (octet_length(location) <= 300),
    ADD CONSTRAINT jobs_source_url_size_check CHECK (octet_length(source_url) <= 2048),
    ADD CONSTRAINT jobs_description_size_check CHECK (octet_length(description) <= 1048576);

CREATE INDEX jobs_workspace_source_url_idx
    ON jobs (workspace_id, source_url)
    WHERE source_url <> '';
