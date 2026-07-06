package ai

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"studyapp/backend/internal/common/metrics"
)

// pricing per 1M tokens (USD). We only track total tokens, so cost is estimated
// with an assumed input/output split.
type pricing struct {
	inputPerM  float64
	outputPerM float64
}

var modelPricing = map[string]pricing{
	modelOpenAIDefault: {inputPerM: 0.15, outputPerM: 0.60},
	modelGeminiDefault: {inputPerM: 0.075, outputPerM: 0.30},
}

// assumed split of total tokens between prompt (input) and completion (output).
const (
	assumedInputShare  = 0.60
	assumedOutputShare = 0.40
)

// EstimateCost returns a USD estimate for the given model and total token count.
func EstimateCost(model string, totalTokens int) float64 {
	p, ok := modelPricing[model]
	if !ok {
		// Default to the cheaper gemini-flash rate to avoid overstating cost.
		p = modelPricing[modelGeminiDefault]
	}
	inputTokens := float64(totalTokens) * assumedInputShare
	outputTokens := float64(totalTokens) * assumedOutputShare
	return (inputTokens/1_000_000)*p.inputPerM + (outputTokens/1_000_000)*p.outputPerM
}

// LogGeneration records a generation attempt in ai_generation_logs. It never
// logs the input text — only word/question counts and usage metadata.
func LogGeneration(
	ctx context.Context,
	db *pgxpool.Pool,
	logger *slog.Logger,
	scanJobID int64,
	provider string,
	model string,
	req GenerateRequest,
	result *GenerateResult,
	genErr error,
) {
	if logger == nil {
		logger = slog.Default()
	}

	status := "success"
	var errMessage *string
	questionCount := 0
	tokensUsed := 0
	var durationMs int64
	usedModel := model

	if genErr != nil {
		status = "failed"
		msg := genErr.Error()
		errMessage = &msg
	}
	if result != nil {
		questionCount = len(result.Questions)
		tokensUsed = result.TokensUsed
		durationMs = result.DurationMs
		if result.ModelUsed != "" {
			usedModel = result.ModelUsed
		}
	}

	cost := EstimateCost(usedModel, tokensUsed)

	if db != nil {
		var jobIDValue any
		if scanJobID > 0 {
			jobIDValue = scanJobID
		}
		_, execErr := db.Exec(ctx, `
			INSERT INTO ai_generation_logs
				(scan_job_id, provider, model_name, question_count, token_usage, duration_ms, cache_hit, cost_estimate, status, error_message, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, false, $7, $8, $9, now())
		`, jobIDValue, provider, usedModel, questionCount, tokensUsed, durationMs, cost, status, errMessage)
		if execErr != nil {
			logger.Error("failed to write ai_generation_logs", "error", execErr)
		}
	}

	// Log metadata only — never the input text.
	logger.Info("ai generation logged",
		"scan_job_id", scanJobID,
		"provider", provider,
		"model", usedModel,
		"word_count", wordCount(req.Text),
		"question_count", questionCount,
		"tokens_used", tokensUsed,
		"duration_ms", durationMs,
		"cost_estimate", cost,
		"status", status,
	)

	metrics.RecordAIGeneration(provider, usedModel, status, time.Duration(durationMs)*time.Millisecond, tokensUsed)
}

func wordCount(text string) int {
	count := 0
	inWord := false
	for _, r := range text {
		if r == ' ' || r == '\n' || r == '\t' || r == '\r' {
			inWord = false
			continue
		}
		if !inWord {
			count++
			inWord = true
		}
	}
	return count
}
