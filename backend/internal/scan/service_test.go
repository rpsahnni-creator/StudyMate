package scan

import (
	"context"
	"errors"
	"testing"
)

type fakeRepository struct {
	consentAccepted      bool
	consentRecorded      bool
	auditLogActions      []string
	findContentCacheResp map[string]any
	createQuizCalls      int
	createQuestionCalls  int
	createContentCalls   int
	generationLogCalls   int
}

func (f *fakeRepository) CreateJob(ctx context.Context, userID int64, req CreateScanJobRequest, initialStatus ScanJobStatus) (ScanJob, error) {
	return ScanJob{ID: 1, UserID: userID, Mode: req.Mode, Status: initialStatus}, nil
}

func (f *fakeRepository) GetJob(ctx context.Context, jobID int64) (ScanJob, error) {
	return ScanJob{ID: jobID}, nil
}

func (f *fakeRepository) GetJobForUser(ctx context.Context, jobID, userID int64) (ScanJob, error) {
	return ScanJob{ID: jobID, UserID: userID}, nil
}

func (f *fakeRepository) CreatePage(ctx context.Context, jobID int64, pageNo int, imageURL string) (ScanPage, error) {
	return ScanPage{ID: 1, ScanJobID: jobID, PageNo: pageNo}, nil
}

func (f *fakeRepository) GetPageByJobAndPageNo(ctx context.Context, jobID int64, pageNo int) (ScanPage, error) {
	return ScanPage{ID: 1, ScanJobID: jobID, PageNo: pageNo}, nil
}

func (f *fakeRepository) EnsurePage(ctx context.Context, jobID int64, pageNo int) (ScanPage, error) {
	return ScanPage{ID: 1, ScanJobID: jobID, PageNo: pageNo}, nil
}

func (f *fakeRepository) UpdatePageChunkMetadata(ctx context.Context, pageID int64, metadata PageChunkMetadata) error {
	return nil
}

func (f *fakeRepository) MarkPageUploaded(ctx context.Context, pageID int64, storageKey string) error {
	return nil
}

func (f *fakeRepository) AllPagesUploaded(ctx context.Context, jobID int64) (bool, error) {
	return true, nil
}

func (f *fakeRepository) UpdateJobStatus(ctx context.Context, jobID int64, status ScanJobStatus, progress int, errMsg *string) error {
	return nil
}

func (f *fakeRepository) UpdateJobQuizID(ctx context.Context, jobID, quizID int64) error {
	return nil
}

func (f *fakeRepository) SaveContentHash(ctx context.Context, jobID int64, contentHash string, pageType string, processed bool) error {
	return nil
}

func (f *fakeRepository) CreateGenerationLog(ctx context.Context, contentHash, modelName, promptVersion string, tokenUsage int, cacheHit bool, costEstimate float64) (int64, error) {
	f.generationLogCalls++
	return 1, nil
}

func (f *fakeRepository) UpdateGenerationLog(ctx context.Context, generationID int64, tokenUsage int, costEstimate float64) error {
	return nil
}

func (f *fakeRepository) CreateQuizRecord(ctx context.Context, chapterID *int64, contentHash string, title string, totalQuestions int) (int64, error) {
	f.createQuizCalls++
	return 10, nil
}

func (f *fakeRepository) CreateQuizRecordWithStatus(ctx context.Context, chapterID *int64, contentHash string, title string, totalQuestions int, status string) (int64, error) {
	f.createQuizCalls++
	return 10, nil
}

func (f *fakeRepository) CreateQuestion(ctx context.Context, chapterID *int64, contentHash, questionText, questionType, sourceType, difficulty string) (int64, error) {
	f.createQuestionCalls++
	return 11, nil
}

func (f *fakeRepository) CreateQuestionWithAnswer(ctx context.Context, chapterID *int64, contentHash, questionText, questionType, sourceType, difficulty, answerStatus string) (int64, error) {
	f.createQuestionCalls++
	return 11, nil
}

func (f *fakeRepository) CreateQuestionExplanation(ctx context.Context, questionID int64, explanation, language string) error {
	return nil
}

func (f *fakeRepository) CreateQuestionOption(ctx context.Context, questionID int64, label, text string, isCorrect bool) error {
	return nil
}

func (f *fakeRepository) UpdateJobPipeline(ctx context.Context, jobID int64, pageType, pipelineText string) error {
	return nil
}

func (f *fakeRepository) UpdateJobStrategy(ctx context.Context, jobID int64, strategy string) error {
	return nil
}

func (f *fakeRepository) UpdateJobChapterSummary(ctx context.Context, jobID int64, summary string) error {
	return nil
}

func (f *fakeRepository) ClearJobPipelineText(ctx context.Context, jobID int64) error {
	return nil
}

func (f *fakeRepository) UpdatePageType(ctx context.Context, pageID int64, pageType string, contentHash string, confidence float64) error {
	return nil
}

func (f *fakeRepository) LinkQuestionToQuiz(ctx context.Context, quizID, questionID int64, orderNo int) error {
	return nil
}

func (f *fakeRepository) CreateContentCache(ctx context.Context, contentHash string, bookID, chapterID *int64, pageNo int, quizID, aiGenerationID *int64) error {
	f.createContentCalls++
	return nil
}

func (f *fakeRepository) FindContentCache(ctx context.Context, contentHash string) (map[string]any, error) {
	if f.findContentCacheResp != nil {
		return f.findContentCacheResp, nil
	}
	return nil, nil
}

func (f *fakeRepository) CreateAuditLog(ctx context.Context, actorUserID *int64, action, entityType, entityID string) error {
	f.consentRecorded = f.consentRecorded || action == "tos.accepted"
	f.auditLogActions = append(f.auditLogActions, action)
	return nil
}

func (f *fakeRepository) HasAcceptedConsent(ctx context.Context, userID int64) (bool, error) {
	return f.consentAccepted, nil
}

func TestCreateJobRequiresConsent(t *testing.T) {
	svc := NewService(&fakeRepository{}, nil, nil)
	_, err := svc.CreateJob(context.Background(), 42, CreateScanJobRequest{Mode: "chapter", SourceText: "sample"})
	if !errors.Is(err, ErrConsentRequired) {
		t.Fatalf("expected ErrConsentRequired, got %v", err)
	}
}

func TestRecordConsentPersistsAuditLog(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo, nil, nil)
	if err := svc.RecordConsent(context.Background(), 7); err != nil {
		t.Fatalf("expected consent recording to succeed, got %v", err)
	}
	if !repo.consentRecorded {
		t.Fatal("expected consent to be recorded in audit logs")
	}
}

func TestCreateJobQueuesForWorker(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo, nil, nil)

	resp, err := svc.CreateJob(context.Background(), 42, CreateScanJobRequest{Mode: "chapter", SourceText: "sample", AcceptedTerms: true})
	if err != nil {
		t.Fatalf("expected queued job creation to succeed, got %v", err)
	}
	if resp.QuizID != nil {
		t.Fatalf("expected no quiz id before worker runs, got %#v", resp.QuizID)
	}
	if resp.Message == "" {
		t.Fatal("expected queued message")
	}
}

func TestCreateJobRecordsEphemeralProcessingAudit(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo, nil, nil)

	_, err := svc.CreateJob(context.Background(), 42, CreateScanJobRequest{Mode: "chapter", SourceText: "sample", AcceptedTerms: true})
	if err != nil {
		t.Fatalf("expected policy-safe creation to succeed, got %v", err)
	}

	found := false
	for _, action := range repo.auditLogActions {
		if action == "content.ephemeral_processed" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected ephemeral processing audit log, got %v", repo.auditLogActions)
	}
}

func TestFallbackGeneratorProducesReadableOutput(t *testing.T) {
	svc := NewService(&fakeRepository{}, nil, nil)
	generated, err := svc.generateQuizContent(context.Background(), "chapter", "Sample chapter content")
	if err != nil {
		t.Fatalf("expected fallback generation to succeed, got %v", err)
	}
	if generated.QuestionText == "" || generated.Explanation == "" {
		t.Fatalf("expected fallback generation to return question and explanation")
	}
}

func TestComputeContentHashNormalizesText(t *testing.T) {
	bookID := int64(10)
	chapterID := int64(20)
	first := computeContentHash(&bookID, &chapterID, 3, "  Hello   World  ")
	second := computeContentHash(&bookID, &chapterID, 3, "hello world")
	if first != second {
		t.Fatalf("expected normalized content to produce the same hash, got %s and %s", first, second)
	}
}
