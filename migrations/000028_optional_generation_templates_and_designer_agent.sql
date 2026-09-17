ALTER TABLE generation_runs ALTER COLUMN template_id DROP NOT NULL;

ALTER TABLE agent_llm_defaults DROP CONSTRAINT agent_llm_defaults_agent_check;
ALTER TABLE agent_llm_defaults
    ADD CONSTRAINT agent_llm_defaults_agent_check
    CHECK (agent IN ('writer', 'template_applier', 'document_designer', 'visual_reviewer', 'job_import', 'job_hunter'));
