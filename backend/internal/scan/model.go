package scan

import (
	"encoding/json"
	"time"
)

type ScanJobStatus string

const (
	ScanJobUploading      ScanJobStatus = "uploading"
	ScanJobPending        ScanJobStatus = "pending"
	ScanJobQueued         ScanJobStatus = "queued" // legacy alias, treated as pending
	ScanJobProcessing     ScanJobStatus = "processing"
	ScanJobOCRComplete    ScanJobStatus = "ocr_complete"
	ScanJobNeedsStrategy  ScanJobStatus = "needs_strategy"
	ScanJobQuizReady      ScanJobStatus = "quiz_ready"
	ScanJobCompleted      ScanJobStatus = "completed" // legacy alias for quiz_ready in API responses
	ScanJobFailed         ScanJobStatus = "failed"
)

type ScanJob struct {
	ID                  int64         `json:"id"`
	UserID              int64         `json:"user_id"`
	BookID              *int64        `json:"book_id,omitempty"`
	ChapterID           *int64        `json:"chapter_id,omitempty"`
	Mode                string        `json:"mode"`
	Status              ScanJobStatus `json:"status"`
	Progress            int           `json:"progress"`
	QuizID              *int64        `json:"quiz_id,omitempty"`
	ErrorMessage        *string       `json:"error_message,omitempty"`
	GenerationStrategy  *string       `json:"generation_strategy,omitempty"`
	DetectedPageType    *string       `json:"detected_page_type,omitempty"`
	ChapterSummary      *string       `json:"chapter_summary,omitempty"`
	ChapterTitle        *string       `json:"chapter_title,omitempty"`
	PipelineText        *string       `json:"-"`
	CreatedAt           time.Time     `json:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at"`
}

type ScanPage struct {
	ID               int64           `json:"id"`
	ScanJobID        int64           `json:"scan_job_id"`
	PageNo           int             `json:"page_no"`
	ImageURL         *string         `json:"image_url,omitempty"`
	PageType         *string         `json:"page_type,omitempty"`
	OCRConfidence    *float64        `json:"ocr_confidence,omitempty"`
	ContentHash      *string         `json:"content_hash,omitempty"`
	Processed        bool            `json:"processed"`
	ChunkMetadata    json.RawMessage `json:"chunk_metadata,omitempty"`
	UploadStatus     string          `json:"upload_status,omitempty"`
	TempStorageKey   *string         `json:"temp_storage_key,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

type CreateScanJobRequest struct {
	BookID              *int64 `json:"book_id,omitempty"`
	ChapterID           *int64 `json:"chapter_id,omitempty"`
	Mode                string `json:"mode"`
	PageNo              int    `json:"page_no,omitempty"`
	SourceText          string `json:"source_text,omitempty"`
	Board               string `json:"board,omitempty"`
	AcceptedTerms       bool   `json:"accepted_terms,omitempty"`
	GenerationStrategy  string `json:"generation_strategy,omitempty"`
	ChapterTitle        string `json:"chapter_title,omitempty"`
}

type ScanJobResponse struct {
	Job     ScanJob               `json:"job"`
	Pages   []ScanPage            `json:"pages,omitempty"`
	QuizID  *int64                `json:"quiz_id,omitempty"`
	Message string                `json:"message,omitempty"`
	Upload  *UploadStatusResponse `json:"upload,omitempty"`
}

type UploadCompleteRequest struct {
	JobID       int64 `json:"jobId"`
	PageNumber  int   `json:"pageNumber"`
	TotalChunks int   `json:"totalChunks"`
}

type UploadCompleteResponse struct {
	JobID  int64  `json:"jobId"`
	Status string `json:"status"`
}

type UploadStatusResponse struct {
	Status           string `json:"status"`
	Progress         int    `json:"progress"`
	RetryCount       int    `json:"retry_count"`
	LastError        string `json:"last_error,omitempty"`
	CanRetry         bool   `json:"can_retry"`
	Message          string `json:"message,omitempty"`
	QuizID           *int64 `json:"quiz_id,omitempty"`
	DetectedPageType string `json:"detected_page_type,omitempty"`
	NeedsStrategy    bool   `json:"needs_strategy,omitempty"`
	ChapterSummary   string `json:"chapter_summary,omitempty"`
}

type SetJobStrategyRequest struct {
	GenerationStrategy string `json:"generation_strategy"`
}

type PageChunkMetadata struct {
	TotalChunks int               `json:"total_chunks"`
	Received    []int             `json:"received"`
	Checksums   map[string]string `json:"checksums"`
}
