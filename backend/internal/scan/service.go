package scan

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"studyapp/backend/internal/scan/storage"
)

type Service struct {
	repo      Repository
	cache     *redis.Client
	generator Generator
	storage   storage.Client
}

// Generator produces quiz content from raw content before it is persisted.
type Generator interface {
	Generate(ctx context.Context, mode, content string) (generatedQuizContent, error)
}

type generatedQuizContent struct {
	QuestionText string
	Explanation  string
	Options      []generatedOption
}

type generatedOption struct {
	Label     string
	Text      string
	IsCorrect bool
}

type defaultGenerator struct{}

func (defaultGenerator) Generate(ctx context.Context, mode, content string) (generatedQuizContent, error) {
	normalizedMode := strings.TrimSpace(mode)
	if normalizedMode == "" {
		normalizedMode = "chapter"
	}
	questionText := fmt.Sprintf("What is the main idea of this %s content?", normalizedMode)
	explanation := fmt.Sprintf("This %s-focused question helps students understand the main idea in simple language without copying textbook wording, and it is written as a fresh study aid.", normalizedMode)
	return generatedQuizContent{
		QuestionText: questionText,
		Explanation:  explanation,
		Options: []generatedOption{
			{Label: "A", Text: "Core idea explained clearly", IsCorrect: true},
			{Label: "B", Text: "A surface detail only", IsCorrect: false},
			{Label: "C", Text: "A repeated textbook phrase", IsCorrect: false},
		},
	}, nil
}

func NewService(repo Repository, cache *redis.Client, store storage.Client) *Service {
	return &Service{repo: repo, cache: cache, generator: defaultGenerator{}, storage: store}
}

// UploadPageImage streams a page image to temp storage and marks the scan page uploaded.
func (s *Service) UploadPageImage(ctx context.Context, jobID int64, pageNo int, reader io.Reader, size int64) error {
	if s.storage == nil {
		return errors.New("storage not configured")
	}
	if size <= 0 || size > int64(storage.MaxPageUploadSize()) {
		return fmt.Errorf("image exceeds max size of %d bytes", storage.MaxPageUploadSize())
	}

	peek := make([]byte, 12)
	n, err := io.ReadFull(reader, peek)
	if err != nil && err != io.ErrUnexpectedEOF {
		return fmt.Errorf("read image header: %w", err)
	}
	rest, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("read image body: %w", err)
	}
	data := append(peek[:n], rest...)
	if int64(len(data)) > int64(storage.MaxPageUploadSize()) {
		return fmt.Errorf("image exceeds max size of %d bytes", storage.MaxPageUploadSize())
	}

	contentType, ext, err := detectImageContentType(data)
	if err != nil {
		return err
	}
	_ = contentType // stored implicitly via extension; PutObject uses raw bytes

	key := fmt.Sprintf("temp/%d/%d/%s%s", jobID, pageNo, uuid.New().String(), ext)
	if err := s.storage.PutObject(ctx, key, data); err != nil {
		return err
	}

	page, err := s.repo.GetPageByJobAndPageNo(ctx, jobID, pageNo)
	if err != nil {
		_ = s.storage.DeleteObject(ctx, key)
		return err
	}
	if err := s.repo.MarkPageUploaded(ctx, page.ID, key); err != nil {
		_ = s.storage.DeleteObject(ctx, key)
		return err
	}
	if ready, err := s.repo.AllPagesUploaded(ctx, jobID); err == nil && ready {
		_ = s.repo.UpdateJobStatus(ctx, jobID, ScanJobPending, 0, nil)
	}
	return nil
}

var (
	ErrConsentRequired         = errors.New("terms of service consent required before scan")
	ErrUnsupportedContentScope = errors.New("content scope not supported; use ncert or state board material")
	ErrJobNotAwaitingStrategy  = errors.New("job is not awaiting generation strategy")
)

func (s *Service) CreateJob(ctx context.Context, userID int64, req CreateScanJobRequest) (ScanJobResponse, error) {
	if err := validateContentScope(req.Board); err != nil {
		return ScanJobResponse{}, err
	}
	if !req.AcceptedTerms {
		accepted, err := s.repo.HasAcceptedConsent(ctx, userID)
		if err != nil {
			return ScanJobResponse{}, err
		}
		if !accepted {
			return ScanJobResponse{}, ErrConsentRequired
		}
	} else if err := s.RecordConsent(ctx, userID); err != nil {
		return ScanJobResponse{}, err
	}
	if err := s.repo.CreateAuditLog(ctx, &userID, "content.ephemeral_processed", "scan_job", strconv.FormatInt(userID, 10)); err != nil {
		return ScanJobResponse{}, err
	}

	job, err := s.repo.CreateJob(ctx, userID, req, initialJobStatus(req))
	if err != nil {
		return ScanJobResponse{}, err
	}

	pageCount := 1
	if req.PageNo > 0 {
		pageCount = req.PageNo
	}

	imageRef := ""
	if req.SourceText != "" {
		imageRef = fmt.Sprintf("stub://inline/%d", job.ID)
	}

	for pageNo := 1; pageNo <= pageCount; pageNo++ {
		if _, err = s.repo.CreatePage(ctx, job.ID, pageNo, imageRef); err != nil {
			return ScanJobResponse{}, err
		}
	}

	return ScanJobResponse{
		Job:     job,
		Message: jobStatusMessage(job.Status),
	}, nil
}

func initialJobStatus(req CreateScanJobRequest) ScanJobStatus {
	if strings.TrimSpace(req.SourceText) != "" {
		return ScanJobPending
	}
	return ScanJobUploading
}

func jobStatusMessage(status ScanJobStatus) string {
	if status == ScanJobUploading {
		return "scan job created; upload image chunks to begin processing"
	}
	return "scan job queued; OCR worker will process shortly"
}

func (s *Service) GetJob(ctx context.Context, jobID int64) (ScanJob, error) {
	return s.repo.GetJob(ctx, jobID)
}

// GetJobForUser fetches a job scoped to its owner, preventing users from
// polling or inspecting scan jobs that belong to someone else.
func (s *Service) GetJobForUser(ctx context.Context, jobID, userID int64) (ScanJob, error) {
	return s.repo.GetJobForUser(ctx, jobID, userID)
}

// SetJobStrategy stores the user's choice for mixed pages and requeues the job.
func (s *Service) SetJobStrategy(ctx context.Context, jobID, userID int64, strategy string) error {
	job, err := s.repo.GetJobForUser(ctx, jobID, userID)
	if err != nil {
		return err
	}
	if job.Status != ScanJobNeedsStrategy {
		return ErrJobNotAwaitingStrategy
	}
	return s.repo.UpdateJobStrategy(ctx, jobID, strategy)
}

func (s *Service) RecordConsent(ctx context.Context, userID int64) error {
	return s.repo.CreateAuditLog(ctx, &userID, "tos.accepted", "user", strconv.FormatInt(userID, 10))
}

func (s *Service) generateQuizContent(ctx context.Context, mode, content string) (generatedQuizContent, error) {
	if s.generator != nil {
		return s.generator.Generate(ctx, mode, content)
	}
	return defaultGenerator{}.Generate(ctx, mode, content)
}

func computeContentHash(bookID, chapterID *int64, pageNo int, rawText string) string {
	normalized := normalizeText(rawText)
	seed := fmt.Sprintf("%v:%v:%d:%s", bookID, chapterID, pageNo, normalized)
	sum := sha256.Sum256([]byte(seed))
	return hex.EncodeToString(sum[:])
}

func normalizeText(raw string) string {
	return strings.ToLower(strings.Join(strings.Fields(raw), " "))
}

func validateContentScope(board string) error {
	if board == "" {
		return nil
	}
	normalized := strings.ToLower(strings.TrimSpace(board))
	switch normalized {
	case "ncert", "state board", "state-board", "stateboard", "cbse", "icse":
		return nil
	default:
		return ErrUnsupportedContentScope
	}
}

func strconvParse(input string) (int64, error) {
	var value int64
	_, err := fmt.Sscanf(input, "%d", &value)
	return value, err
}
