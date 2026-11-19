ALTER TABLE job_applications
    ALTER COLUMN interview_at TYPE TIMESTAMP USING interview_at AT TIME ZONE 'UTC',
    ALTER COLUMN created_at   TYPE TIMESTAMP USING created_at   AT TIME ZONE current_setting('TIMEZONE');
