-- Repair databases that ran migration 14 before its unsupported down section
-- was removed. The statements are intentionally idempotent for fresh installs.
ALTER TABLE templates ADD COLUMN IF NOT EXISTS preview_object_id text;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'templates_preview_object_unique'
          AND conrelid = 'templates'::regclass
    ) THEN
        ALTER TABLE templates
            ADD CONSTRAINT templates_preview_object_unique UNIQUE (workspace_id, preview_object_id);
    END IF;
END
$$;

UPDATE templates
SET state = 'needs_user_action',
    error_code = 'preview_missing',
    error_message = 'Re-upload the template source to generate its PDF preview.',
    updated_at = now()
WHERE state = 'ready'
  AND preview_object_id IS NULL;
