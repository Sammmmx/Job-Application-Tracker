-- interview_at and created_at were TIMESTAMP (no time zone), which silently drops the offset
-- of any value sent with one, e.g. 14:30+05:30 would be stored as 14:30 and read back as UTC.
-- TIMESTAMPTZ stores the real instant.
--
-- Existing interview_at values were written from UTC input, so they are read as UTC.
-- Existing created_at values came from now() in the server's time zone, so they are read in it.
ALTER TABLE job_applications
    ALTER COLUMN interview_at TYPE TIMESTAMPTZ USING interview_at AT TIME ZONE 'UTC',
    ALTER COLUMN created_at   TYPE TIMESTAMPTZ USING created_at   AT TIME ZONE current_setting('TIMEZONE');
