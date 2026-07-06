-- Resumable chunked upload support for scan pages
ALTER TABLE scan_pages
    ADD COLUMN IF NOT EXISTS chunk_metadata JSONB NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS upload_status TEXT NOT NULL DEFAULT 'pending',
    ADD COLUMN IF NOT EXISTS temp_storage_key TEXT;

CREATE INDEX IF NOT EXISTS idx_scan_pages_upload_status ON scan_pages (scan_job_id, upload_status);
