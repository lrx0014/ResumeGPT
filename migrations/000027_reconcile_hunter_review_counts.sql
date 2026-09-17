UPDATE job_hunters hunter
SET last_found_count = GREATEST(0, hunter.last_found_count - review.failed_count),
    updated_at = now()
FROM (
    SELECT workspace_id, hunter_id, count(*)::integer AS failed_count
    FROM job_hunter_review_items
    WHERE failure_code = 'legacy_import_failed'
    GROUP BY workspace_id, hunter_id
) review
WHERE hunter.workspace_id = review.workspace_id
  AND hunter.id = review.hunter_id;
