package ai

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"studyapp/backend/internal/devstub"
)

// StubGenerator returns deterministic, coherent quizzes for tests, CI, and local dev.
type StubGenerator struct{}

func NewStubGenerator() *StubGenerator { return &StubGenerator{} }

func (g *StubGenerator) ProviderName() string { return ProviderStub }

func (g *StubGenerator) ModelName() string { return "stub-v3" }

func (g *StubGenerator) Generate(ctx context.Context, req GenerateRequest) (*GenerateResult, error) {
	_ = ctx
	mix := req.effectiveMix()
	if g.ProviderName() == ProviderStub && mix.Total() > 13 && req.Mix.Total() == 0 && req.ScanMode != ScanModeExistingQuestions {
		// Smaller default for dev unless caller set explicit mix or AI_FULL_MIX=1
		mix = DevChapterMix()
	}
	if req.ScanMode == ScanModeExistingQuestions {
		mix = DefaultExistingMix()
		if mix.MCQ > 10 {
			mix.MCQ = 10
		}
	}

	topics := devstub.TopicsMatchingText(req.Text)
	if len(topics) == 0 {
		topics = []devstub.Topic{devstub.TopicForSeed(req.Text)}
	}

	sum := sha256.Sum256([]byte(req.Text))
	questions := make([]GeneratedQuestion, 0, mix.Total())
	idx := 0

	for i := 0; i < mix.MCQ; i++ {
		questions = append(questions, g.mcqQuestion(topics, sum, idx, req))
		idx++
	}
	for i := 0; i < mix.FillBlank; i++ {
		questions = append(questions, g.fillBlankQuestion(topics, sum, idx, req))
		idx++
	}
	for i := 0; i < mix.TrueFalse; i++ {
		questions = append(questions, g.trueFalseQuestion(topics, sum, idx, req))
		idx++
	}

	summary := ""
	if req.WantSummary && req.ScanMode == ScanModeChapter {
		t := topics[0]
		summary = fmt.Sprintf("%s ke baare mein: %s (Simple Hindi summary — stub)", t.Name, t.Paragraph)
	}

	return &GenerateResult{
		Questions:      questions,
		ChapterSummary: summary,
		TokensUsed:     0,
		ModelUsed:      g.ModelName(),
		DurationMs:     0,
	}, nil
}

func (g *StubGenerator) mcqQuestion(topics []devstub.Topic, sum [32]byte, i int, req GenerateRequest) GeneratedQuestion {
	seed := binary.BigEndian.Uint32(sum[(i*4)%28 : (i*4)%28+4])
	topic := topics[i%len(topics)]
	correctIndex := int(seed % 4)
	labels := []string{"A", "B", "C", "D"}
	options := make([]GeneratedOption, 4)
	wrongIdx := 0
	for j := 0; j < 4; j++ {
		text := topic.Correct
		if j != correctIndex {
			text = topic.Distractors[wrongIdx]
			wrongIdx++
		}
		options[j] = GeneratedOption{Label: labels[j], Text: text}
	}
	return GeneratedQuestion{
		Text:         topic.Question,
		Type:         QuestionTypeMCQ,
		Options:      options,
		CorrectIndex: correctIndex,
		Explanation:  devstub.SimpleHindiExplanation(topic),
		Difficulty:   req.difficulty(),
		Topic:        topic.Name,
	}
}

func (g *StubGenerator) fillBlankQuestion(topics []devstub.Topic, sum [32]byte, i int, req GenerateRequest) GeneratedQuestion {
	seed := binary.BigEndian.Uint32(sum[(i*4)%28 : (i*4)%28+4])
	topic := topics[i%len(topics)]
	correctIndex := int(seed % 4)
	labels := []string{"A", "B", "C", "D"}
	options := make([]GeneratedOption, 4)
	wrongIdx := 0
	for j := 0; j < 4; j++ {
		text := topic.Correct
		if j != correctIndex {
			text = topic.Distractors[wrongIdx]
			wrongIdx++
		}
		options[j] = GeneratedOption{Label: labels[j], Text: text}
	}
	return GeneratedQuestion{
		Text:         devstub.FillBlankText(topic),
		Type:         QuestionTypeFillBlank,
		Options:      options,
		CorrectIndex: correctIndex,
		Explanation:  devstub.SimpleHindiExplanation(topic),
		Difficulty:   req.difficulty(),
		Topic:        topic.Name,
	}
}

func (g *StubGenerator) trueFalseQuestion(topics []devstub.Topic, sum [32]byte, i int, req GenerateRequest) GeneratedQuestion {
	topic := topics[i%len(topics)]
	seed := binary.BigEndian.Uint32(sum[(i*4)%28 : (i*4)%28+4])
	isTrue := seed%2 == 0
	text := devstub.TrueStatement(topic)
	correctIndex := 0
	if !isTrue {
		text = devstub.FalseStatement(topic)
		correctIndex = 1
	}
	return GeneratedQuestion{
		Text: text,
		Type: QuestionTypeTrueFalse,
		Options: []GeneratedOption{
			{Label: "A", Text: "True"},
			{Label: "B", Text: "False"},
		},
		CorrectIndex: correctIndex,
		Explanation:  devstub.SimpleHindiExplanation(topic),
		Difficulty:   req.difficulty(),
		Topic:        topic.Name,
	}
}
