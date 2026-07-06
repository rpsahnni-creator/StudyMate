package scan

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateJob(ctx context.Context, userID int64, req CreateScanJobRequest, initialStatus ScanJobStatus) (ScanJob, error)
	GetJob(ctx context.Context, jobID int64) (ScanJob, error)
	GetJobForUser(ctx context.Context, jobID, userID int64) (ScanJob, error)
	CreatePage(ctx context.Context, jobID int64, pageNo int, imageURL string) (ScanPage, error)
	GetPageByJobAndPageNo(ctx context.Context, jobID int64, pageNo int) (ScanPage, error)
	EnsurePage(ctx context.Context, jobID int64, pageNo int) (ScanPage, error)
	UpdatePageChunkMetadata(ctx context.Context, pageID int64, metadata PageChunkMetadata) error
	MarkPageUploaded(ctx context.Context, pageID int64, storageKey string) error
	AllPagesUploaded(ctx context.Context, jobID int64) (bool, error)
	UpdateJobStatus(ctx context.Context, jobID int64, status ScanJobStatus, progress int, errMsg *string) error
	UpdateJobQuizID(ctx context.Context, jobID, quizID int64) error
	SaveContentHash(ctx context.Context, jobID int64, contentHash string, pageType string, processed bool) error
	CreateGenerationLog(ctx context.Context, contentHash, modelName, promptVersion string, tokenUsage int, cacheHit bool, costEstimate float64) (int64, error)
	UpdateGenerationLog(ctx context.Context, generationID int64, tokenUsage int, costEstimate float64) error
	CreateQuizRecord(ctx context.Context, chapterID *int64, contentHash string, title string, totalQuestions int) (int64, error)
	CreateQuestion(ctx context.Context, chapterID *int64, contentHash, questionText, questionType, sourceType, difficulty string) (int64, error)
	CreateQuestionOption(ctx context.Context, questionID int64, label, text string, isCorrect bool) error
	CreateQuestionExplanation(ctx context.Context, questionID int64, explanation, language string) error
	LinkQuestionToQuiz(ctx context.Context, quizID, questionID int64, orderNo int) error
	UpdateJobPipeline(ctx context.Context, jobID int64, pageType, pipelineText string) error
	UpdateJobStrategy(ctx context.Context, jobID int64, strategy string) error
	UpdateJobChapterSummary(ctx context.Context, jobID int64, summary string) error
	ClearJobPipelineText(ctx context.Context, jobID int64) error
	UpdatePageType(ctx context.Context, pageID int64, pageType string, contentHash string, confidence float64) error
	CreateContentCache(ctx context.Context, contentHash string, bookID, chapterID *int64, pageNo int, quizID, aiGenerationID *int64) error
	FindContentCache(ctx context.Context, contentHash string) (map[string]any, error)
	HasAcceptedConsent(ctx context.Context, userID int64) (bool, error)
	CreateAuditLog(ctx context.Context, actorUserID *int64, action, entityType, entityID string) error
}

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

func scanJobColumns() string {
	return `id, user_id, book_id, chapter_id, mode, status, progress, quiz_id, error_message,
		generation_strategy, detected_page_type, chapter_summary, chapter_title, pipeline_text, created_at, updated_at`
}

func scanJob(dest *ScanJob) []any {
	return []any{
		&dest.ID, &dest.UserID, &dest.BookID, &dest.ChapterID, &dest.Mode, &dest.Status, &dest.Progress,
		&dest.QuizID, &dest.ErrorMessage, &dest.GenerationStrategy, &dest.DetectedPageType,
		&dest.ChapterSummary, &dest.ChapterTitle, &dest.PipelineText, &dest.CreatedAt, &dest.UpdatedAt,
	}
}

func (r *postgresRepository) CreateJob(ctx context.Context, userID int64, req CreateScanJobRequest, initialStatus ScanJobStatus) (ScanJob, error) {
	if initialStatus == "" {
		initialStatus = ScanJobPending
	}
	mode := normalizeScanMode(req.Mode)
	var strategy *string
	if s := strings.TrimSpace(req.GenerationStrategy); s != "" {
		strategy = &s
	}
	var chapterTitle *string
	if t := strings.TrimSpace(req.ChapterTitle); t != "" {
		chapterTitle = &t
	}
	var job ScanJob
	err := r.pool.QueryRow(ctx, `
		INSERT INTO scan_jobs (user_id, book_id, chapter_id, mode, status, progress, generation_strategy, chapter_title, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), now())
		RETURNING `+scanJobColumns(),
		userID, req.BookID, req.ChapterID, mode, initialStatus, 0, strategy, chapterTitle).Scan(scanJob(&job)...)
	if err != nil {
		return ScanJob{}, err
	}
	return job, nil
}

func (r *postgresRepository) GetJob(ctx context.Context, jobID int64) (ScanJob, error) {
	var job ScanJob
	err := r.pool.QueryRow(ctx, `
		SELECT `+scanJobColumns()+`
		FROM scan_jobs WHERE id = $1
	`, jobID).Scan(scanJob(&job)...)
	if err != nil {
		return ScanJob{}, err
	}
	return job, nil
}

func (r *postgresRepository) GetJobForUser(ctx context.Context, jobID, userID int64) (ScanJob, error) {
	var job ScanJob
	err := r.pool.QueryRow(ctx, `
		SELECT `+scanJobColumns()+`
		FROM scan_jobs WHERE id = $1 AND user_id = $2
	`, jobID, userID).Scan(scanJob(&job)...)
	if err != nil {
		return ScanJob{}, err
	}
	return job, nil
}

func scanPageColumns() string {
	return `id, scan_job_id, page_no, image_url, page_type, ocr_confidence, content_hash, processed,
		COALESCE(chunk_metadata, '{}'::jsonb), COALESCE(upload_status, 'pending'), temp_storage_key, created_at`
}

func scanPage(dest *ScanPage) []any {
	return []any{
		&dest.ID, &dest.ScanJobID, &dest.PageNo, &dest.ImageURL, &dest.PageType, &dest.OCRConfidence,
		&dest.ContentHash, &dest.Processed, &dest.ChunkMetadata, &dest.UploadStatus, &dest.TempStorageKey, &dest.CreatedAt,
	}
}

func (r *postgresRepository) CreatePage(ctx context.Context, jobID int64, pageNo int, imageURL string) (ScanPage, error) {
	var page ScanPage
	err := r.pool.QueryRow(ctx, `
		INSERT INTO scan_pages (scan_job_id, page_no, image_url, processed, upload_status, created_at)
		VALUES ($1, $2, $3, false, 'pending', now())
		RETURNING `+scanPageColumns(),
		jobID, pageNo, nullIfEmpty(imageURL)).Scan(scanPage(&page)...)
	if err != nil {
		return ScanPage{}, err
	}
	return page, nil
}

func (r *postgresRepository) UpdateJobStatus(ctx context.Context, jobID int64, status ScanJobStatus, progress int, errMsg *string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE scan_jobs SET status = $1, progress = $2, error_message = $3, updated_at = now() WHERE id = $4
	`, status, progress, errMsg, jobID)
	return err
}

func (r *postgresRepository) UpdateJobQuizID(ctx context.Context, jobID, quizID int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE scan_jobs SET quiz_id = $1, updated_at = now() WHERE id = $2
	`, quizID, jobID)
	return err
}

func (r *postgresRepository) SaveContentHash(ctx context.Context, jobID int64, contentHash string, pageType string, processed bool) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE scan_pages SET content_hash = $1, page_type = $2, processed = $3 WHERE scan_job_id = $4
	`, contentHash, pageType, processed, jobID)
	return err
}

func (r *postgresRepository) CreateQuizRecord(ctx context.Context, chapterID *int64, contentHash string, title string, totalQuestions int) (int64, error) {
	var quizID int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO quizzes (chapter_id, content_hash, title, total_questions, generation_type, created_at)
		VALUES ($1, $2, $3, $4, $5, now()) RETURNING id
	`, chapterID, contentHash, title, totalQuestions, "ai").Scan(&quizID)
	if err != nil {
		return 0, err
	}
	return quizID, nil
}

func (r *postgresRepository) CreateQuestion(ctx context.Context, chapterID *int64, contentHash, questionText, questionType, sourceType, difficulty string) (int64, error) {
	if strings.TrimSpace(questionType) == "" {
		questionType = "mcq"
	}
	if strings.TrimSpace(sourceType) == "" {
		sourceType = "ai_generated"
	}
	if strings.TrimSpace(difficulty) == "" {
		difficulty = "medium"
	}
	var questionID int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO questions (chapter_id, content_hash, question_type, question_text, difficulty, source_type, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now()) RETURNING id
	`, chapterID, contentHash, questionType, questionText, difficulty, sourceType, "active").Scan(&questionID)
	if err != nil {
		return 0, err
	}
	return questionID, nil
}

func (r *postgresRepository) CreateQuestionOption(ctx context.Context, questionID int64, label, text string, isCorrect bool) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO question_options (question_id, option_label, option_text, is_correct, created_at)
		VALUES ($1, $2, $3, $4, now())
	`, questionID, label, text, isCorrect)
	return err
}

func (r *postgresRepository) CreateQuestionExplanation(ctx context.Context, questionID int64, explanation, language string) error {
	if strings.TrimSpace(language) == "" {
		language = "hi"
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO question_explanations (question_id, style, explanation_text, language, created_at)
		VALUES ($1, $2, $3, $4, now())
	`, questionID, "simple", explanation, language)
	return err
}

func (r *postgresRepository) UpdateJobPipeline(ctx context.Context, jobID int64, pageType, pipelineText string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE scan_jobs
		SET detected_page_type = $1, pipeline_text = $2, updated_at = now()
		WHERE id = $3
	`, pageType, pipelineText, jobID)
	return err
}

func (r *postgresRepository) UpdateJobStrategy(ctx context.Context, jobID int64, strategy string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE scan_jobs
		SET generation_strategy = $1, status = $2, progress = 10, updated_at = now()
		WHERE id = $3
	`, strategy, ScanJobPending, jobID)
	return err
}

func (r *postgresRepository) UpdateJobChapterSummary(ctx context.Context, jobID int64, summary string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE scan_jobs SET chapter_summary = $1, updated_at = now() WHERE id = $2
	`, summary, jobID)
	return err
}

func (r *postgresRepository) ClearJobPipelineText(ctx context.Context, jobID int64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE scan_jobs SET pipeline_text = NULL, updated_at = now() WHERE id = $1
	`, jobID)
	return err
}

func (r *postgresRepository) UpdatePageType(ctx context.Context, pageID int64, pageType string, contentHash string, confidence float64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE scan_pages
		SET content_hash = $1, ocr_confidence = $2, processed = true, page_type = $3
		WHERE id = $4
	`, contentHash, confidence, pageType, pageID)
	return err
}

func (r *postgresRepository) LinkQuestionToQuiz(ctx context.Context, quizID, questionID int64, orderNo int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO quiz_questions (quiz_id, question_id, order_no, created_at)
		VALUES ($1, $2, $3, now())
	`, quizID, questionID, orderNo)
	return err
}

func (r *postgresRepository) CreateContentCache(ctx context.Context, contentHash string, bookID, chapterID *int64, pageNo int, quizID, aiGenerationID *int64) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO content_cache (book_id, chapter_id, page_no, content_hash, page_type, generated_quiz_id, ai_generation_id, hit_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 1, now())
		ON CONFLICT (content_hash) DO UPDATE SET hit_count = content_cache.hit_count + 1
	`, bookID, chapterID, pageNo, contentHash, "chapter", quizID, aiGenerationID)
	return err
}

func (r *postgresRepository) FindContentCache(ctx context.Context, contentHash string) (map[string]any, error) {
	var quizID *int64
	var pageType string
	err := r.pool.QueryRow(ctx, `
		SELECT generated_quiz_id, page_type FROM content_cache WHERE content_hash = $1
	`, contentHash).Scan(&quizID, &pageType)
	if err != nil {
		return nil, err
	}
	return map[string]any{"quiz_id": quizID, "page_type": pageType}, nil
}

func (r *postgresRepository) HasAcceptedConsent(ctx context.Context, userID int64) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM audit_logs WHERE actor_user_id = $1 AND action = 'tos.accepted'
		)
	`, userID).Scan(&exists)
	return exists, err
}

func (r *postgresRepository) CreateGenerationLog(ctx context.Context, contentHash, modelName, promptVersion string, tokenUsage int, cacheHit bool, costEstimate float64) (int64, error) {
	var generationID int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO ai_generation_logs (flag_key, content_hash, model_name, prompt_version, token_usage, cache_hit, cost_estimate, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now()) RETURNING id
	`, "scan_quiz_module", contentHash, modelName, promptVersion, tokenUsage, cacheHit, costEstimate).Scan(&generationID)
	if err != nil {
		return 0, err
	}
	return generationID, nil
}

func (r *postgresRepository) UpdateGenerationLog(ctx context.Context, generationID int64, tokenUsage int, costEstimate float64) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE ai_generation_logs SET token_usage = $1, cost_estimate = $2 WHERE id = $3
	`, tokenUsage, costEstimate, generationID)
	return err
}

func (r *postgresRepository) CreateJobEvent(ctx context.Context, jobID string, eventType string, message string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO job_events (job_id, event_type, message, created_at) VALUES ($1, $2, $3, now())
	`, jobID, eventType, message)
	return err
}

func (r *postgresRepository) CreateAuditLog(ctx context.Context, actorUserID *int64, action, entityType, entityID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO audit_logs (actor_user_id, action, entity_type, entity_id, created_at) VALUES ($1, $2, $3, $4, now())
	`, actorUserID, action, entityType, entityID)
	return err
}

func (r *postgresRepository) CreateContentFlag(ctx context.Context, questionID int64, reason string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO content_flags (question_id, reason, status, created_at) VALUES ($1, $2, 'pending', now())
	`, questionID, reason)
	return err
}

func (r *postgresRepository) CreateAttempt(ctx context.Context, quizID, userID int64, score float64, correct, wrong, skipped int, status string) (int64, error) {
	var attemptID int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO quiz_attempts (quiz_id, user_id, score, correct_count, wrong_count, skipped_count, status, started_at, submitted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now(), now()) RETURNING id
	`, quizID, userID, score, correct, wrong, skipped, status).Scan(&attemptID)
	if err != nil {
		return 0, err
	}
	return attemptID, nil
}

func (r *postgresRepository) CreateAttemptAnswer(ctx context.Context, attemptID, questionID, selectedOptionID int64, isCorrect bool) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO quiz_attempt_answers (attempt_id, question_id, selected_option_id, is_correct, answered_at)
		VALUES ($1, $2, $3, $4, now())
	`, attemptID, questionID, selectedOptionID, isCorrect)
	return err
}

func (r *postgresRepository) CreateReport(ctx context.Context, attemptID int64, accuracy float64, weakTopics string, summary string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO quiz_reports (attempt_id, accuracy, weak_topics_json, summary_text, created_at)
		VALUES ($1, $2, $3, $4, now())
	`, attemptID, accuracy, weakTopics, summary)
	return err
}

func (r *postgresRepository) CreateSubscription(ctx context.Context, userID int64, planID int64, status string) (int64, error) {
	var subscriptionID int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO subscriptions (user_id, plan_id, status, provider, starts_at, ends_at)
		VALUES ($1, $2, $3, $4, now(), now() + interval '1 month') RETURNING id
	`, userID, planID, status, "razorpay").Scan(&subscriptionID)
	if err != nil {
		return 0, err
	}
	return subscriptionID, nil
}

func (r *postgresRepository) CreatePayment(ctx context.Context, userID, subscriptionID int64, amount float64, status string) (int64, error) {
	var paymentID int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO payments (user_id, subscription_id, amount, status, provider, transaction_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, now()) RETURNING id
	`, userID, subscriptionID, amount, status, "razorpay", fmt.Sprintf("txn-%d", userID)).Scan(&paymentID)
	if err != nil {
		return 0, err
	}
	return paymentID, nil
}

func (r *postgresRepository) CreatePlan(ctx context.Context, name string, monthlyPrice float64, scanLimit int) (int64, error) {
	var planID int64
	err := r.pool.QueryRow(ctx, `
		INSERT INTO plans (name, price_monthly, scan_limit, features_json, active, created_at)
		VALUES ($1, $2, $3, $4, true, now()) RETURNING id
	`, name, monthlyPrice, scanLimit, `{"free":false}`).Scan(&planID)
	if err != nil {
		return 0, err
	}
	return planID, nil
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func (r *postgresRepository) GetPageByJobAndPageNo(ctx context.Context, jobID int64, pageNo int) (ScanPage, error) {
	var page ScanPage
	err := r.pool.QueryRow(ctx, `
		SELECT `+scanPageColumns()+`
		FROM scan_pages
		WHERE scan_job_id = $1 AND page_no = $2
	`, jobID, pageNo).Scan(scanPage(&page)...)
	if err != nil {
		return ScanPage{}, err
	}
	return page, nil
}

func (r *postgresRepository) EnsurePage(ctx context.Context, jobID int64, pageNo int) (ScanPage, error) {
	page, err := r.GetPageByJobAndPageNo(ctx, jobID, pageNo)
	if err == nil {
		return page, nil
	}
	return r.CreatePage(ctx, jobID, pageNo, "")
}

func (r *postgresRepository) UpdatePageChunkMetadata(ctx context.Context, pageID int64, metadata PageChunkMetadata) error {
	raw, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE scan_pages SET chunk_metadata = $1, upload_status = 'uploading' WHERE id = $2
	`, raw, pageID)
	return err
}

func (r *postgresRepository) MarkPageUploaded(ctx context.Context, pageID int64, storageKey string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE scan_pages
		SET temp_storage_key = $1,
		    image_url = $1,
		    upload_status = 'uploaded',
		    chunk_metadata = '{}'::jsonb
		WHERE id = $2
	`, storageKey, pageID)
	return err
}

func (r *postgresRepository) AllPagesUploaded(ctx context.Context, jobID int64) (bool, error) {
	var pending int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM scan_pages
		WHERE scan_job_id = $1 AND upload_status <> 'uploaded'
	`, jobID).Scan(&pending)
	if err != nil {
		return false, err
	}
	var total int
	err = r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM scan_pages WHERE scan_job_id = $1`, jobID).Scan(&total)
	if err != nil {
		return false, err
	}
	return total > 0 && pending == 0, nil
}
