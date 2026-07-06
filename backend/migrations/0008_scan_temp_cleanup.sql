-- Soft-delete marker for orphaned temp storage cleanup
ALTER TABLE scan_pages
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_scan_pages_temp_cleanup
    ON scan_pages (created_at)
    WHERE temp_storage_key IS NOT NULL AND deleted_at IS NULL;
