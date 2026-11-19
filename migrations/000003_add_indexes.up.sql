-- Matches the list query: ORDER BY applied_at DESC, id DESC
CREATE INDEX IF NOT EXISTS idx_job_applications_applied_at
    ON job_applications (applied_at DESC, id DESC);

-- The status filter compares lower(status), so the index has to be on lower(status) too.
-- A plain index on status would be ignored by that query.
CREATE INDEX IF NOT EXISTS idx_job_applications_status_lower
    ON job_applications (lower(status));
