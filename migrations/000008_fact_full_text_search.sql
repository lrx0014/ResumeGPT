CREATE INDEX fact_versions_statement_fts_idx ON fact_versions
    USING gin (to_tsvector('simple', statement));
