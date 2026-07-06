package ai

import "context"

// VisionImage is ephemeral page bytes — never logged or persisted.
type VisionImage struct {
	Bytes []byte
	MIME  string
}

// VisionRequest combines quiz generation params with one or more page images.
type VisionRequest struct {
	GenerateRequest
	Images []VisionImage
}

// VisionGenerator extracts or generates quiz questions directly from document images.
type VisionGenerator interface {
	GenerateFromImages(ctx context.Context, req VisionRequest) (*GenerateResult, error)
	ProviderName() string
	ModelName() string
}
