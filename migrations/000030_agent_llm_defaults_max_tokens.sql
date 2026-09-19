ALTER TABLE agent_llm_defaults
    ADD COLUMN max_tokens integer
        CHECK (max_tokens IS NULL OR (max_tokens >= 1 AND max_tokens <= 32768));
