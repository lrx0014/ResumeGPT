ALTER TABLE workspace_settings
    DROP CONSTRAINT IF EXISTS workspace_settings_interface_language_check;

ALTER TABLE workspace_settings
    ADD CONSTRAINT workspace_settings_interface_language_check
    CHECK (interface_language IN ('en', 'de', 'fr', 'es', 'ja', 'zh-CN', 'zh-TW'));
