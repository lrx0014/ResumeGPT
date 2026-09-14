ALTER TABLE templates ADD COLUMN preview_object_id text;
ALTER TABLE templates ADD CONSTRAINT templates_preview_object_unique UNIQUE (workspace_id, preview_object_id);

-- Existing templates predate stored previews. Re-uploading their source will
-- create both extracted text and a PDF preview through the normal worker flow.
UPDATE templates
SET state = 'needs_user_action',
    error_code = 'preview_missing',
    error_message = 'Re-upload the template source to generate its PDF preview.',
    updated_at = now()
WHERE state = 'ready';
