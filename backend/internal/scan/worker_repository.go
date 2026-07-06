package scan

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// WorkerRepository extends scan persistence for background job processing.
type WorkerRepository interface {
	Repository
	ClaimPendingJobs(ctx context.Context, limit int) ([]ScanJob, error)
	ListPagesByJobID(ctx context.Context, jobID int64) ([]ScanPage, error)
	UpdatePageProcessed(ctx context.Context, pageID int64, contentHash string, confidence float64, pageType string) error
}

type workerRepository struct {
	Repository
	pool *pgxpool.Pool
}

// NewWorkerRepository wraps the base repository with worker queries.
func NewWorkerRepository(base Repository, pool *pgxpool.Pool) WorkerRepository {
	return &workerRepository{Repository: base, pool: pool}
}

func (r *workerRepository) ClaimPendingJobs(ctx context.Context, limit int) ([]ScanJob, error) {
	if limit <= 0 {
		limit = 1
	}
	rows, err := r.pool.Query(ctx, `
		WITH picked AS (
			SELECT id
			FROM scan_jobs
			WHERE status IN ('pending', 'queued')
			   OR (status = 'processing' AND updated_at < NOW() - INTERVAL '30 minutes')
			ORDER BY created_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE scan_jobs AS j
		SET status = 'processing', progress = 5, updated_at = now()
		FROM picked
		WHERE j.id = picked.id
		RETURNING j.id, j.user_id, j.book_id, j.chapter_id, j.mode, j.status, j.progress, j.error_message,
		          j.generation_strategy, j.detected_page_type, j.chapter_summary, j.chapter_title, j.pipeline_text, j.created_at, j.updated_at
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []ScanJob
	for rows.Next() {
		var job ScanJob
		if err := rows.Scan(
			&job.ID, &job.UserID, &job.BookID, &job.ChapterID, &job.Mode, &job.Status, &job.Progress, &job.ErrorMessage,
			&job.GenerationStrategy, &job.DetectedPageType, &job.ChapterSummary, &job.ChapterTitle, &job.PipelineText, &job.CreatedAt, &job.UpdatedAt,
		); err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *workerRepository) ListPagesByJobID(ctx context.Context, jobID int64) ([]ScanPage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, scan_job_id, page_no, image_url, page_type, ocr_confidence, content_hash, processed,
		       COALESCE(chunk_metadata, '{}'::jsonb), COALESCE(upload_status, 'pending'), temp_storage_key, created_at
		FROM scan_pages
		WHERE scan_job_id = $1
		ORDER BY page_no ASC
	`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pages []ScanPage
	for rows.Next() {
		var page ScanPage
		if err := rows.Scan(
			&page.ID, &page.ScanJobID, &page.PageNo, &page.ImageURL, &page.PageType, &page.OCRConfidence,
			&page.ContentHash, &page.Processed, &page.ChunkMetadata, &page.UploadStatus, &page.TempStorageKey, &page.CreatedAt,
		); err != nil {
			return nil, err
		}
		pages = append(pages, page)
	}
	if len(pages) == 0 {
		return nil, fmt.Errorf("no scan pages for job %d", jobID)
	}
	return pages, rows.Err()
}

func (r *workerRepository) UpdatePageProcessed(ctx context.Context, pageID int64, contentHash string, confidence float64, pageType string) error {
	return r.UpdatePageType(ctx, pageID, pageType, contentHash, confidence)
}
