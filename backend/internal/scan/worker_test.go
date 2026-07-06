package scan

import (
	"context"
	"testing"

	"studyapp/backend/internal/scan/ocr"
	"studyapp/backend/internal/scan/storage"
)

type workerFakeRepo struct {
	fakeRepository
	job         ScanJob
	pages       []ScanPage
	statusTrail []ScanJobStatus
}

func (w *workerFakeRepo) GetJob(ctx context.Context, jobID int64) (ScanJob, error) {
	return w.job, nil
}

func (w *workerFakeRepo) ListPagesByJobID(ctx context.Context, jobID int64) ([]ScanPage, error) {
	return w.pages, nil
}

func (w *workerFakeRepo) UpdateJobStatus(ctx context.Context, jobID int64, status ScanJobStatus, progress int, errMsg *string) error {
	w.statusTrail = append(w.statusTrail, status)
	w.job.Status = status
	w.job.Progress = progress
	return nil
}

func (w *workerFakeRepo) UpdatePageProcessed(ctx context.Context, pageID int64, contentHash string, confidence float64, pageType string) error {
	for i := range w.pages {
		if w.pages[i].ID == pageID {
			w.pages[i].ContentHash = &contentHash
			w.pages[i].Processed = true
		}
	}
	return nil
}

func (w *workerFakeRepo) ClaimPendingJobs(ctx context.Context, limit int) ([]ScanJob, error) {
	return nil, nil
}

func TestWorkerProcessJobStatusTransitions(t *testing.T) {
	image := "temp://page-1"
	repo := &workerFakeRepo{
		job: ScanJob{ID: 7, UserID: 42, Mode: "chapter", Status: ScanJobProcessing},
		pages: []ScanPage{{
			ID: 1, ScanJobID: 7, PageNo: 1, ImageURL: &image,
		}},
	}

	provider, err := ocr.NewProvider(ocr.OCRConfig{Provider: ocr.ProviderStub})
	if err != nil {
		t.Fatalf("provider init failed: %v", err)
	}

	worker := NewWorker(
		repo,
		nil,
		nil,
		storage.NewLocalClient(),
		provider,
		nil,
		nil,
		nil,
		nil,
		WorkerConfig{MinConfidence: 0.5, MaxPagesPerJob: 10},
	)

	if err := worker.ProcessJob(context.Background(), 7); err != nil {
		t.Fatalf("process job failed: %v", err)
	}

	want := []ScanJobStatus{ScanJobOCRComplete, ScanJobQuizReady}
	if len(repo.statusTrail) < len(want) {
		t.Fatalf("expected at least %d status updates, got %v", len(want), repo.statusTrail)
	}
	last := repo.statusTrail[len(repo.statusTrail)-1]
	if last != ScanJobQuizReady {
		t.Fatalf("expected final status quiz_ready, got %s", last)
	}
}

func TestWorkerSkipsNonPendingJob(t *testing.T) {
	repo := &workerFakeRepo{
		job: ScanJob{ID: 1, Status: ScanJobQuizReady},
	}
	provider, _ := ocr.NewProvider(ocr.OCRConfig{Provider: ocr.ProviderStub})
	worker := NewWorker(repo, nil, nil, storage.NewLocalClient(), provider, nil, nil, nil, nil, WorkerConfig{})
	if err := worker.ProcessJob(context.Background(), 1); err != nil {
		t.Fatalf("expected skip without error, got %v", err)
	}
	if len(repo.statusTrail) != 0 {
		t.Fatalf("expected no status updates, got %v", repo.statusTrail)
	}
}
