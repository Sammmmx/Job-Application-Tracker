UPDATE job_applications SET job_url  = '' WHERE job_url  IS NULL;
UPDATE job_applications SET job_type = '' WHERE job_type IS NULL;
UPDATE job_applications SET notes    = '' WHERE notes    IS NULL;

ALTER TABLE job_applications
    ALTER COLUMN job_url  SET DEFAULT '',
    ALTER COLUMN job_url  SET NOT NULL,
    ALTER COLUMN job_type SET DEFAULT '',
    ALTER COLUMN job_type SET NOT NULL,
    ALTER COLUMN notes    SET DEFAULT '',
    ALTER COLUMN notes    SET NOT NULL;