// Temporary end-to-end check for the Gemini vision quiz pipeline.
// Run: go run ./cmd/e2evision <image-path>
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"studyapp/backend/internal/quiz/ai"
)

func main() {
	_ = godotenv.Load(".env")

	imgPath := "_e2e_page.jpg"
	if len(os.Args) > 1 {
		imgPath = os.Args[1]
	}
	data, err := os.ReadFile(imgPath)
	if err != nil {
		log.Fatalf("read image: %v", err)
	}
	fmt.Printf("image: %s (%d bytes)\n", imgPath, len(data))

	cfg := ai.LoadConfig()
	fmt.Printf("provider=%s model_override=%s key_set=%v\n", cfg.Provider, cfg.ModelOverride, cfg.GeminiKey != "")

	gen, err := ai.NewGeminiVisionGenerator(cfg)
	if err != nil {
		log.Fatalf("init vision generator: %v", err)
	}
	fmt.Printf("using model: %s\n", gen.ModelName())

	req := ai.VisionRequest{
		GenerateRequest: ai.GenerateRequest{
			Board:              "ncert",
			Subject:            "chapter",
			Chapter:            "Photosynthesis in Plants",
			ScanMode:           ai.ScanModeChapter,
			FlexibleVision:     true,
			VisionMaxQuestions: ai.VisionMaxQuestionsDefault(),
			WantSummary:        true,
			Difficulty:         "medium",
			Language:           "hindi",
			Rules:              ai.DefaultExplanationRules(),
		},
		Images: []ai.VisionImage{{Bytes: data, MIME: ai.MIMEFromObjectRef(imgPath)}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	start := time.Now()
	res, err := gen.GenerateFromImages(ctx, req)
	if err != nil {
		log.Fatalf("GENERATE FAILED: %v", err)
	}
	fmt.Printf("\n=== SUCCESS in %s | model=%s tokens=%d questions=%d ===\n",
		time.Since(start), res.ModelUsed, res.TokensUsed, len(res.Questions))
	if res.ChapterSummary != "" {
		fmt.Printf("\nChapter summary:\n%s\n", res.ChapterSummary)
	}
	for i, q := range res.Questions {
		fmt.Printf("\nQ%d [%s]: %s\n", i+1, q.Type, q.Text)
		for j, o := range q.Options {
			marker := " "
			if j == q.CorrectIndex {
				marker = "*"
			}
			fmt.Printf("   %s %s. %s\n", marker, o.Label, o.Text)
		}
		fmt.Printf("   explanation: %s\n", q.Explanation)
	}
}
