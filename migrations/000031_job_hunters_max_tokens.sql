ALTER TABLE job_hunters
    ADD COLUMN max_tokens integer
        CHECK (max_tokens IS NULL OR (max_tokens >= 1 AND max_tokens <= 32768));
