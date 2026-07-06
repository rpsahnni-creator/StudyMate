package scan

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"studyapp/backend/internal/common/metrics"
	"studyapp/backend/internal/quiz/ai"
	"studyapp/backend/internal/scan/ocr"
	"studyapp/backend/internal/scan/storage"
)

const (
	workerPollInterval  = 5 * time.Second
	workerMaxConcurrent = 5
	minQuestionsDefault = 5
)

// errRateLimited signals the per-user AI rate cap was hit; the job is requeued.
var errRateLimited = errors.New("ai rate limit exceeded")

// Notifier emits side effects when a job completes.
type Notifier interface {
	QuizReady(ctx context.Context, userID, jobID, quizID int64) error
}

type slogNotifier struct {
	logger *slog.Logger
}

func (n slogNotifier) QuizReady(ctx context.Context, userID, jobID, quizID int64) error {
	logger := n.logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Info("quiz ready notification queued",
		"user_id", userID,
		"job_id", jobID,
		"quiz_id", quizID,
		"category", "quiz_ready",
	)
	return nil
}

// WorkerConfig configures polling and OCR limits.
type WorkerConfig struct {
	PollInterval    time.Duration
	MaxConcurrent   int
	MaxPagesPerJob  int
	MinConfidence   float64
	AIQuestionCount int
}

type pageOutcome struct {
	hash      string
	objectRef string
}

// Worker processes pending scan jobs asynchronously.
type Worker struct {
	repo         WorkerRepository
	db           *pgxpool.Pool
	cacheService *CacheService
	storage      storage.Client
	ocr          ocr.Provider
	gen          ai.Generator
	rateLimiter  *ai.RateLimiter
	notifier     Notifier
	logger       *slog.Logger
	cfg          WorkerConfig
	sem          chan struct{}
	inFlight     sync.WaitGroup
}

func NewWorker(
	repo WorkerRepository,
	db *pgxpool.Pool,
	redisClient *redis.Client,
	store storage.Client,
	ocrProvider ocr.Provider,
	generator ai.Generator,
	notifier Notifier,
	logger *slog.Logger,
	cacheService *CacheService,
	cfg WorkerConfig,
) *Worker {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = workerPollInterval
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = workerMaxConcurrent
	}
	if logger == nil {
		logger = slog.Default()
	}
	if generator == nil {
		generator = ai.NewStubGenerator()
	}
	if notifier == nil {
		notifier = slogNotifier{logger: logger}
	}
	if cacheService == nil && (redisClient != nil || db != nil) {
		cacheService = NewCacheService(redisClient, db, logger)
	}
	return &Worker{
		repo:         repo,
		db:           db,
		cacheService: cacheService,
		storage:      store,
		ocr:          ocrProvider,
		gen:          generator,
		rateLimiter:  ai.NewRateLimiter(0, 0),
		notifier:     notifier,
		logger:       logger,
		cfg:          cfg,
		sem:          make(chan struct{}, cfg.MaxConcurrent),
	}
}

func (w *Worker) Run(ctx context.Context) error {
	w.logger.Info("scan worker started",
		"ocr_provider", w.ocr.Name(),
		"poll_interval", w.cfg.PollInterval.String(),
		"max_concurrent", w.cfg.MaxConcurrent,
	)
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("scan worker shutting down, waiting for in-flight jobs")
			w.inFlight.Wait()
			w.logger.Info("scan worker stopped")
			return ctx.Err()
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

func (w *Worker) poll(ctx context.Context) {
	jobs, err := w.repo.ClaimPendingJobs(ctx, w.cfg.MaxConcurrent)
	if err != nil {
		w.logger.Error("claim pending jobs failed", "error", err)
		return
	}
	var wg sync.WaitGroup
	for _, job := range jobs {
		wg.Add(1)
		w.inFlight.Add(1)
		go func(job ScanJob) {
			defer wg.Done()
			defer w.inFlight.Done()
			w.sem <- struct{}{}
			defer func() { <-w.sem }()
			jobCtx := context.WithoutCancel(ctx)
			if err := w.ProcessJob(jobCtx, job.ID); err != nil {
				w.logger.Error("process job failed", "job_id", job.ID, "error", err)
			}
		}(job)
	}
	wg.Wait()
}

// ProcessJob runs OCR and quiz generation for a single job.
func (w *Worker) ProcessJob(ctx context.Context, jobID int64) error {
	start := time.Now()
	ocrName := w.ocr.Name()
	defer func() {
		metrics.ObserveScanJobDuration(ocrName, time.Since(start))
	}()

	job, err := w.repo.GetJob(ctx, jobID)
	if err != nil {
		return err
	}
	if job.Status == ScanJobQuizReady || job.Status == ScanJobFailed || job.Status == ScanJobNeedsStrategy {
		return nil
	}
	if !isPendingStatus(job.Status) && job.Status != ScanJobProcessing {
		w.logger.Debug("skip job", "job_id", jobID, "status", job.Status)
		return nil
	}

	if job.Status != ScanJobProcessing {
		if err := w.repo.UpdateJobStatus(ctx, jobID, ScanJobProcessing, 10, nil); err != nil {
			return err
		}
	}

	var combinedText string
	var outcomes []pageOutcome
	resumeFromPipeline := job.PipelineText != nil && strings.TrimSpace(*job.PipelineText) != ""

	if resumeFromPipeline {
		combinedText = *job.PipelineText
		pages, err := w.repo.ListPagesByJobID(ctx, jobID)
		if err != nil {
			return w.failJob(ctx, jobID, err)
		}
		for _, page := range pages {
			outcomes = append(outcomes, pageOutcome{objectRef: pageObjectRef(page)})
		}
	} else {
		pages, err := w.repo.ListPagesByJobID(ctx, jobID)
		if err != nil {
			return w.failJob(ctx, jobID, err)
		}
		if w.cfg.MaxPagesPerJob > 0 && len(pages) > w.cfg.MaxPagesPerJob {
			return w.failJob(ctx, jobID, fmt.Errorf("job exceeds OCR_MAX_PAGES_PER_JOB limit (%d)", w.cfg.MaxPagesPerJob))
		}

		ephemeralTexts := make([]string, 0, len(pages))
		for _, page := range pages {
			text, outcome, err := w.processPage(ctx, job, page)
			if err != nil {
				w.logger.Warn("page ocr failed", "job_id", jobID, "page_id", page.ID, "page_no", page.PageNo, "error", err)
				continue
			}
			ephemeralTexts = append(ephemeralTexts, text)
			outcomes = append(outcomes, outcome)
		}
		if len(outcomes) == 0 {
			return w.failJob(ctx, jobID, fmt.Errorf("all pages failed OCR"))
		}
		combinedText = strings.Join(ephemeralTexts, "\n")
	}

	if err := w.repo.UpdateJobStatus(ctx, jobID, ScanJobOCRComplete, 50, nil); err != nil {
		return err
	}

	pageType := DetectPageType(combinedText)
	strategy := derefStr(job.GenerationStrategy)
	effectiveMode := ResolveGenerationMode(job.Mode, pageType, strategy)

	if pageType == ai.PageTypeMixed && effectiveMode == "" {
		if err := w.repo.UpdateJobPipeline(ctx, jobID, pageType, combinedText); err != nil {
			return w.failJob(ctx, jobID, err)
		}
		if err := w.repo.UpdateJobStatus(ctx, jobID, ScanJobNeedsStrategy, 45, nil); err != nil {
			return err
		}
		w.logger.Info("mixed page detected, awaiting user strategy", "job_id", jobID, "page_type", pageType)
		return nil
	}

	contentHash := ContentCacheHash(combinedText)
	skipCache := w.gen.ProviderName() == ai.ProviderStub

	if w.cacheService != nil && !skipCache && !resumeFromPipeline {
		cached, hit, err := w.cacheService.Lookup(ctx, contentHash)
		if err != nil {
			return w.failJob(ctx, jobID, fmt.Errorf("cache lookup failed: %w", err))
		}
		if hit && cached != nil {
			metrics.RecordScanCacheHit()
			w.deletePageObjects(ctx, outcomes)
			combinedText = ""
			w.cacheService.LogCacheHit(ctx, contentHash, jobID, w.gen.ProviderName(), cached)
			if err := w.repo.UpdateJobQuizID(ctx, jobID, cached.QuizID); err != nil {
				return err
			}
			_ = w.repo.ClearJobPipelineText(ctx, jobID)
			if err := w.repo.UpdateJobStatus(ctx, jobID, ScanJobQuizReady, 100, nil); err != nil {
				return err
			}
			metrics.RecordScanJob("completed")
			return w.notifier.QuizReady(ctx, job.UserID, jobID, cached.QuizID)
		}
		metrics.RecordScanCacheMiss()
	}

	quizID, summary, err := w.persistQuiz(ctx, job, jobID, contentHash, combinedText, pageType, effectiveMode)
	combinedText = ""
	if errors.Is(err, errRateLimited) {
		w.logger.Warn("ai rate limit hit, requeueing job", "user_id", job.UserID, "job_id", jobID)
		_ = w.repo.UpdateJobStatus(ctx, jobID, ScanJobPending, 40, nil)
		return nil
	}
	if err != nil {
		return w.failJob(ctx, jobID, err)
	}

	w.deletePageObjects(ctx, outcomes)
	_ = w.repo.ClearJobPipelineText(ctx, jobID)
	if summary != "" {
		_ = w.repo.UpdateJobChapterSummary(ctx, jobID, summary)
	}
	if err := w.repo.UpdateJobStatus(ctx, jobID, ScanJobQuizReady, 100, nil); err != nil {
		return err
	}
	metrics.RecordScanJob("completed")
	return w.notifier.QuizReady(ctx, job.UserID, jobID, quizID)
}

func (w *Worker) persistQuiz(ctx context.Context, job ScanJob, jobID int64, aggregateHash, ephemeralText, pageType, effectiveMode string) (int64, string, error) {
	if w.rateLimiter != nil && !w.rateLimiter.Allow(job.UserID) {
		return 0, "", errRateLimited
	}

	mix := ai.DefaultChapterMix()
	sourceType := ai.SourceAIGenerated
	wantSummary := effectiveMode == ai.ScanModeChapter
	if effectiveMode == ai.ScanModeExistingQuestions {
		mix = ai.DefaultExistingMix()
		sourceType = ai.SourceScannedExisting
		wantSummary = false
	}
	if w.gen.ProviderName() == ai.ProviderStub && effectiveMode == ai.ScanModeChapter {
		mix = ai.DevChapterMix()
	}

	minQuestions := mix.Total()
	if minQuestions <= 0 {
		minQuestions = minQuestionsDefault
	}

	req := ai.GenerateRequest{
		Text:        ephemeralText,
		Subject:     job.Mode,
		ScanMode:    effectiveMode,
		PageType:    pageType,
		Mix:         mix,
		WantSummary: wantSummary,
		Difficulty:  "medium",
		Language:    "hindi",
		Rules:       ai.DefaultExplanationRules(),
	}

	var (
		result *ai.GenerateResult
		genErr error
	)
	for attempt := 0; attempt < 2; attempt++ {
		result, genErr = w.gen.Generate(ctx, req)
		if genErr == nil && result != nil && len(result.Questions) >= minQuestions {
			break
		}
		if genErr == nil && (result == nil || len(result.Questions) < minQuestions) {
			got := 0
			if result != nil {
				got = len(result.Questions)
			}
			genErr = fmt.Errorf("ai returned only %d questions (min %d)", got, minQuestions)
		}
	}

	ai.LogGeneration(ctx, w.db, w.logger, jobID, w.gen.ProviderName(), w.gen.ModelName(), req, result, genErr)

	if genErr != nil {
		return 0, "", genErr
	}

	title := fmt.Sprintf("Quiz for job %d", jobID)
	if effectiveMode == ai.ScanModeExistingQuestions {
		title = fmt.Sprintf("Extracted quiz for job %d", jobID)
	}

	quizID, err := w.repo.CreateQuizRecord(ctx, job.ChapterID, aggregateHash, title, len(result.Questions))
	if err != nil {
		return 0, "", err
	}

	for i, question := range result.Questions {
		qType := question.Type
		if qType == "" {
			qType = ai.QuestionTypeMCQ
		}
		questionID, err := w.repo.CreateQuestion(ctx, job.ChapterID, aggregateHash, question.Text, qType, sourceType, question.Difficulty)
		if err != nil {
			return 0, "", err
		}
		for idx, option := range question.Options {
			isCorrect := idx == question.CorrectIndex
			label := option.Label
			if label == "" {
				label = string(rune('A' + idx))
			}
			if err := w.repo.CreateQuestionOption(ctx, questionID, label, option.Text, isCorrect); err != nil {
				return 0, "", err
			}
		}
		if err := w.repo.CreateQuestionExplanation(ctx, questionID, question.Explanation, "hi"); err != nil {
			return 0, "", err
		}
		if err := w.repo.LinkQuestionToQuiz(ctx, quizID, questionID, i+1); err != nil {
			return 0, "", err
		}
	}

	if w.cacheService != nil {
		modelUsed := result.ModelUsed
		if modelUsed == "" {
			modelUsed = w.gen.ModelName()
		}
		if err := w.cacheService.Store(ctx, aggregateHash, CacheMeta{
			Board:         "",
			Subject:       effectiveMode,
			Chapter:       chapterLabel(job.ChapterID),
			QuizID:        quizID,
			AIProvider:    w.gen.ProviderName(),
			ModelUsed:     modelUsed,
			TokensUsed:    result.TokensUsed,
			GenerationMs:  result.DurationMs,
			QuestionCount: len(result.Questions),
			ScanJobID:     jobID,
			BookID:        job.BookID,
			ChapterID:     job.ChapterID,
			PageNo:        0,
		}); err != nil {
			return 0, "", err
		}
	}
	if err := w.repo.UpdateJobQuizID(ctx, jobID, quizID); err != nil {
		return 0, "", err
	}
	return quizID, result.ChapterSummary, nil
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ocrCtxForPage(ctx context.Context, job ScanJob) context.Context {
	ocrCtx := ocr.ContextWithJobID(ctx, job.ID)
	if title := derefStr(job.ChapterTitle); title != "" {
		ocrCtx = ocr.ContextWithChapterTitle(ocrCtx, title)
	}
	return ocrCtx
}

func chapterLabel(chapterID *int64) string {
	if chapterID == nil {
		return ""
	}
	return strconv.FormatInt(*chapterID, 10)
}

func (w *Worker) processPage(ctx context.Context, job ScanJob, page ScanPage) (string, pageOutcome, error) {
	var zero pageOutcome

	objectRef := pageObjectRef(page)
	fetchURL, err := w.storage.PresignedURL(ctx, objectRef)
	if err != nil {
		return "", zero, err
	}

	var result ocr.Result
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		result, lastErr = ocr.ExtractResult(ocrCtxForPage(ctx, job), w.ocr, fetchURL)
		if lastErr == nil {
			break
		}
	}
	if lastErr != nil {
		return "", zero, lastErr
	}
	if w.cfg.MinConfidence > 0 && result.Confidence < w.cfg.MinConfidence {
		return "", zero, fmt.Errorf("ocr confidence %.2f below minimum %.2f", result.Confidence, w.cfg.MinConfidence)
	}

	w.logger.Info("ocr page complete",
		"job_id", job.ID,
		"page_id", page.ID,
		"page_no", page.PageNo,
		"provider", w.ocr.Name(),
		"word_count", result.WordCount,
		"confidence", result.Confidence,
	)

	ephemeralText := result.RawText
	pageType := DetectPageType(ephemeralText)
	hash := computeContentHash(job.BookID, job.ChapterID, page.PageNo, ephemeralText)
	if err := w.repo.UpdatePageProcessed(ctx, page.ID, hash, result.Confidence, pageType); err != nil {
		ephemeralText = ""
		result.RawText = ""
		return "", zero, err
	}

	outcome := pageOutcome{hash: hash, objectRef: objectRef}
	result.RawText = ""
	return ephemeralText, outcome, nil
}

func (w *Worker) deletePageObjects(ctx context.Context, outcomes []pageOutcome) {
	for _, outcome := range outcomes {
		if outcome.objectRef == "" {
			continue
		}
		_ = w.storage.DeleteObject(ctx, outcome.objectRef)
	}
}

func (w *Worker) failJob(ctx context.Context, jobID int64, err error) error {
	metrics.RecordScanJob("failed")
	pages, listErr := w.repo.ListPagesByJobID(ctx, jobID)
	if listErr == nil {
		var refs []pageOutcome
		for _, page := range pages {
			refs = append(refs, pageOutcome{objectRef: pageObjectRef(page)})
		}
		w.deletePageObjects(ctx, refs)
	}
	msg := err.Error()
	_ = w.repo.UpdateJobStatus(ctx, jobID, ScanJobFailed, 0, &msg)
	return err
}

func pageObjectRef(page ScanPage) string {
	if page.TempStorageKey != nil && *page.TempStorageKey != "" {
		return *page.TempStorageKey
	}
	if page.ImageURL != nil {
		return *page.ImageURL
	}
	return ""
}

func isPendingStatus(status ScanJobStatus) bool {
	return status == ScanJobPending || status == ScanJobQueued
}
