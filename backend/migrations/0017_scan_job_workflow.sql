-- Product workflow: page-type detection, mixed-page strategy, chapter summary
ALTER TABLE scan_jobs
    ADD COLUMN IF NOT EXISTS generation_strategy TEXT,
    ADD COLUMN IF NOT EXISTS detected_page_type TEXT,
    ADD COLUMN IF NOT EXISTS chapter_summary TEXT,
    ADD COLUMN IF NOT EXISTS pipeline_text TEXT;
