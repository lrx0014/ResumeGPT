ALTER TABLE generation_steps DROP CONSTRAINT generation_steps_kind_check;

ALTER TABLE generation_steps
    ADD CONSTRAINT generation_steps_kind_check
    CHECK (kind IN ('writer_draft', 'rendered_pdf', 'reviewer_feedback', 'user_prompt', 'system_warning', 'configuration_change'));
