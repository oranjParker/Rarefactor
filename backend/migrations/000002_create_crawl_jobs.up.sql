CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

DO $$ BEGIN
CREATE TYPE job_status AS ENUM (
        'PENDING',
        'RUNNING',
        'PAUSED',
        'COMPLETED',
        'FAILED',
        'CANCELLED'
    );
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE IF NOT EXISTS crawl_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    seed_url TEXT NOT NULL,
    max_depth INT NOT NULL DEFAULT 2,
    crawl_mode TEXT NOT NULL,
    namespace TEXT NOT NULL,

    status job_status NOT NULL DEFAULT 'PENDING',
    worker_id TEXT,

    pages_crawled INT DEFAULT 0,
    errors_count INT DEFAULT 0,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    error_message TEXT,
    checkpoint_data JSONB
    );

CREATE INDEX IF NOT EXISTS idx_crawl_jobs_status ON crawl_jobs(status);
CREATE INDEX IF NOT EXISTS idx_crawl_jobs_created_at ON crawl_jobs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_crawl_jobs_namespace ON crawl_jobs(namespace);
CREATE INDEX IF NOT EXISTS idx_crawl_jobs_seed_url ON crawl_jobs(seed_url);

CREATE OR REPLACE FUNCTION update_crawl_jobs_updated_at_func()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_crawl_jobs_updated_at
    BEFORE UPDATE ON crawl_jobs
    FOR EACH ROW
    EXECUTE FUNCTION update_crawl_jobs_updated_at_func();