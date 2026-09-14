ALTER TABLE templates
    ADD COLUMN entry_file text NOT NULL DEFAULT ''
    CHECK (octet_length(entry_file) <= 300);
