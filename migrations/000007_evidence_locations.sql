ALTER TABLE source_segments
    ADD COLUMN page_number integer CHECK (page_number > 0),
    ADD COLUMN confidence real NOT NULL DEFAULT 1 CHECK (confidence BETWEEN 0 AND 1),
    ADD COLUMN bounding_box jsonb CHECK (
        bounding_box IS NULL OR (
            jsonb_typeof(bounding_box) = 'object'
            AND bounding_box ?& ARRAY['x','y','width','height']
        )
    );
