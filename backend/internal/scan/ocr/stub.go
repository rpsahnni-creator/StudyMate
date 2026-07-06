package ocr

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"studyapp/backend/internal/devstub"
)

// StubProvider returns deterministic educational OCR text for tests and local dev.
type StubProvider struct {
	minConfidence float64
}

func NewStubProvider() *StubProvider {
	return &StubProvider{minConfidence: 0.95}
}

func (s *StubProvider) Name() string { return ProviderStub }

func (s *StubProvider) ExtractText(ctx context.Context, imageURL string) (string, error) {
	result, err := s.extractResult(ctx, imageURL)
	if err != nil {
		return "", err
	}
	return result.RawText, nil
}

func (s *StubProvider) extractResult(ctx context.Context, imageURL string) (Result, error) {
	seed := imageURL
	if jobID, ok := jobIDFromContext(ctx); ok {
		pageNo := pageNoFromObjectRef(imageURL)
		if pageNo > 0 {
			seed = fmt.Sprintf("job:%d:page:%d", jobID, pageNo)
		} else {
			seed = fmt.Sprintf("job:%d", jobID)
		}
	}
	var text string
	if hint := chapterTitleFromContext(ctx); hint != "" {
		text = devstub.ParagraphForChapterHint(hint, seed)
	} else {
		text = devstub.ParagraphForSeed(seed)
	}
	if pageNo := pageNoFromObjectRef(imageURL); pageNo > 0 {
		text = fmt.Sprintf("%s (scan page %d)", text, pageNo)
	}
	return buildResult(text, s.minConfidence, "eng+hin"), nil
}

func pageNoFromObjectRef(ref string) int {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return 0
	}
	parts := strings.Split(strings.Trim(ref, "/"), "/")
	for i, part := range parts {
		if part != "temp" || i+2 >= len(parts) {
			continue
		}
		if n, err := strconv.Atoi(parts[i+2]); err == nil && n > 0 {
			return n
		}
	}
	return 0
}
