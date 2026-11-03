CREATE TABLE IF NOT EXISTS job_applications (
    id           SERIAL PRIMARY KEY,
    company      VARCHAR(150) NOT NULL,
    role         VARCHAR(150) NOT NULL,
    job_url      TEXT,
    status       VARCHAR(30) NOT NULL DEFAULT 'Applied',
    job_type     VARCHAR(30),
    applied_at   DATE NOT NULL DEFAULT CURRENT_DATE,
    interview_at TIMESTAMP,
    notes        TEXT,
    created_at   TIMESTAMP NOT NULL DEFAULT now()
);