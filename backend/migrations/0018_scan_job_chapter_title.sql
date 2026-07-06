ALTER TABLE scan_jobs
    ADD COLUMN IF NOT EXISTS chapter_title TEXT;
