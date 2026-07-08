package quiz

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"

	"studyapp/backend/internal/quiz/ai"
)

// ownsQuiz reports whether userID created the scan job that produced quizID.
func (s *Service) ownsQuiz(ctx context.Context, quizID, userID int64) (bool, error) {
	var exists bool
	err := s.db.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM scan_jobs WHERE quiz_id = $1 AND user_id = $2)
	`, quizID, userID).Scan(&exists)
	return exists, err
}

// GetDraft returns the editable draft exam (with answers) for its owner.
func (s *Service) GetDraft(ctx context.Context, quizID, userID int64) (*DraftDetail, error) {
	owns, err := s.ownsQuiz(ctx, quizID, userID)
	if err != nil {
		return nil, err
	}
	if !owns {
		return nil, ErrQuizForbidden
	}

	var detail DraftDetail
	err = s.db.QueryRow(ctx, `
		SELECT id, title, status, total_questions FROM quizzes WHERE id = $1
	`, quizID).Scan(&detail.ID, &detail.Title, &detail.Status, &detail.TotalQuestions)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrQuizNotFound
	}
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT ques.id, ques.question_text, ques.question_type, ques.answer_status,
		       COALESCE(
		         (SELECT e.explanation_text FROM question_explanations e WHERE e.question_id = ques.id ORDER BY e.id ASC LIMIT 1),
		         ''
		       ) AS explanation
		FROM quiz_questions qq
		JOIN questions ques ON ques.id = qq.question_id
		WHERE qq.quiz_id = $1
		ORDER BY qq.order_no ASC
	`, quizID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	index := make(map[int64]int)
	for rows.Next() {
		var q DraftQuestion
		if err := rows.Scan(&q.ID, &q.Text, &q.Type, &q.AnswerStatus, &q.Explanation); err != nil {
			return nil, err
		}
		q.Options = []DraftOption{}
		index[q.ID] = len(detail.Questions)
		detail.Questions = append(detail.Questions, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(detail.Questions) > 0 {
		optRows, err := s.db.Query(ctx, `
			SELECT o.question_id, o.id, o.option_label, o.option_text, o.is_correct
			FROM question_options o
			JOIN quiz_questions qq ON qq.question_id = o.question_id
			WHERE qq.quiz_id = $1
			ORDER BY o.question_id, o.option_label ASC
		`, quizID)
		if err != nil {
			return nil, err
		}
		defer optRows.Close()
		for optRows.Next() {
			var questionID int64
			var opt DraftOption
			if err := optRows.Scan(&questionID, &opt.ID, &opt.Label, &opt.Text, &opt.IsCorrect); err != nil {
				return nil, err
			}
			if idx, ok := index[questionID]; ok {
				detail.Questions[idx].Options = append(detail.Questions[idx].Options, opt)
			}
		}
		if err := optRows.Err(); err != nil {
			return nil, err
		}
	}

	for _, q := range detail.Questions {
		if q.AnswerStatus == ai.AnswerStatusUnknown {
			detail.NeedsAnswerCount++
		}
	}
	return &detail, nil
}

// SaveDraft replaces the full question list of a draft exam with the reviewer's
// edits. It is only allowed while the quiz is still a draft.
func (s *Service) SaveDraft(ctx context.Context, quizID, userID int64, req SaveDraftRequest) (*DraftDetail, error) {
	owns, err := s.ownsQuiz(ctx, quizID, userID)
	if err != nil {
		return nil, err
	}
	if !owns {
		return nil, ErrQuizForbidden
	}

	var status string
	var chapterID *int64
	var contentHash *string
	err = s.db.QueryRow(ctx, `SELECT status, chapter_id, content_hash FROM quizzes WHERE id = $1`, quizID).
		Scan(&status, &chapterID, &contentHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrQuizNotFound
	}
	if err != nil {
		return nil, err
	}
	if status != ai.QuizStatusDraft {
		return nil, ErrQuizNotDraft
	}

	normalized, err := validateDraftInput(req.Questions)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := deleteQuizQuestions(ctx, tx, quizID); err != nil {
		return nil, err
	}

	hash := ""
	if contentHash != nil {
		hash = *contentHash
	}
	for i, q := range normalized {
		if err := insertDraftQuestion(ctx, tx, quizID, chapterID, hash, q, i+1); err != nil {
			return nil, err
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE quizzes SET total_questions = $1 WHERE id = $2`, len(normalized), quizID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return s.GetDraft(ctx, quizID, userID)
}

// PublishDraft validates that every question has an answer, generates any missing
// explanations, and flips the quiz to published so students can attempt it.
func (s *Service) PublishDraft(ctx context.Context, quizID, userID int64) (*PublishResult, error) {
	draft, err := s.GetDraft(ctx, quizID, userID)
	if err != nil {
		return nil, err
	}
	if draft.Status != ai.QuizStatusDraft {
		return nil, ErrQuizNotDraft
	}
	if len(draft.Questions) == 0 {
		return nil, fmt.Errorf("%w: exam has no questions", ErrDraftInvalid)
	}

	// Every question must have exactly one correct option selected.
	var missing []int64
	for _, q := range draft.Questions {
		if correctOption(q.Options) == nil {
			missing = append(missing, q.ID)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: %d question(s) still need an answer", ErrAnswersIncomplete, len(missing))
	}

	// Generate explanations for questions that don't have one yet.
	for _, q := range draft.Questions {
		if strings.TrimSpace(q.Explanation) != "" {
			continue
		}
		explanation := s.explainForQuestion(ctx, q)
		if strings.TrimSpace(explanation) == "" {
			continue
		}
		if _, err := s.db.Exec(ctx, `
			INSERT INTO question_explanations (question_id, style, explanation_text, language, created_at)
			VALUES ($1, 'simple', $2, $3, now())
		`, q.ID, explanation, ai.ExplanationDBCode(ai.DetectContentLanguage(q.Text))); err != nil {
			return nil, err
		}
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `UPDATE quizzes SET status = 'published' WHERE id = $1`, quizID); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE questions SET answer_status = 'set'
		WHERE id IN (SELECT question_id FROM quiz_questions WHERE quiz_id = $1)
	`, quizID); err != nil {
		return nil, err
	}
	// Mark the scan job complete so the app stops showing "needs review".
	if _, err := tx.Exec(ctx, `
		UPDATE scan_jobs SET status = 'quiz_ready', progress = 100, updated_at = now()
		WHERE quiz_id = $1 AND user_id = $2
	`, quizID, userID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &PublishResult{QuizID: quizID, Status: "published", TotalQuestions: len(draft.Questions)}, nil
}

func (s *Service) explainForQuestion(ctx context.Context, q DraftQuestion) string {
	correct := correctOption(q.Options)
	if correct == nil {
		return ""
	}
	if s.explainer == nil {
		return ""
	}
	opts := make([]ai.GeneratedOption, 0, len(q.Options))
	for _, o := range q.Options {
		opts = append(opts, ai.GeneratedOption{Label: o.Label, Text: o.Text})
	}
	explanation, err := s.explainer.Explain(ctx, ai.ExplainRequest{
		QuestionText: q.Text,
		QuestionType: q.Type,
		Options:      opts,
		CorrectLabel: correct.Label,
		CorrectText:  correct.Text,
		Language:     ai.DetectContentLanguage(q.Text),
	})
	if err != nil {
		s.logger.Warn("explain question failed", "question_id", q.ID, "error", err)
		return ""
	}
	return explanation
}

func correctOption(options []DraftOption) *DraftOption {
	for i := range options {
		if options[i].IsCorrect {
			return &options[i]
		}
	}
	return nil
}

// normalizedDraftQuestion is a validated question ready to persist.
type normalizedDraftQuestion struct {
	Text         string
	Type         string
	AnswerStatus string
	Explanation  string
	CorrectIndex int
	Options      []DraftOptionInput
}

func validateDraftInput(questions []DraftQuestionInput) ([]normalizedDraftQuestion, error) {
	if len(questions) == 0 {
		return nil, fmt.Errorf("%w: at least one question is required", ErrDraftInvalid)
	}
	out := make([]normalizedDraftQuestion, 0, len(questions))
	for i, q := range questions {
		text := strings.TrimSpace(q.Text)
		if text == "" {
			return nil, fmt.Errorf("%w: question %d has empty text", ErrDraftInvalid, i+1)
		}
		qType := strings.TrimSpace(q.Type)
		minOpts, maxOpts := 4, 4
		switch qType {
		case ai.QuestionTypeMCQ:
			minOpts, maxOpts = 2, 6
		case ai.QuestionTypeFillBlank:
			minOpts, maxOpts = 2, 4
		case ai.QuestionTypeTrueFalse:
			minOpts, maxOpts = 2, 2
		default:
			return nil, fmt.Errorf("%w: question %d has invalid type %q", ErrDraftInvalid, i+1, q.Type)
		}

		opts := make([]DraftOptionInput, 0, len(q.Options))
		for _, o := range q.Options {
			ot := strings.TrimSpace(o.Text)
			if ot == "" {
				continue
			}
			label := strings.TrimSpace(o.Label)
			if label == "" {
				label = string(rune('A' + len(opts)))
			}
			opts = append(opts, DraftOptionInput{Label: label, Text: ot})
		}
		if len(opts) < minOpts || len(opts) > maxOpts {
			return nil, fmt.Errorf("%w: question %d (%s) must have %d-%d options, got %d", ErrDraftInvalid, i+1, qType, minOpts, maxOpts, len(opts))
		}

		correctIndex := q.CorrectIndex
		answerStatus := ai.AnswerStatusSet
		if correctIndex < 0 || correctIndex >= len(opts) {
			correctIndex = ai.CorrectIndexUnknown
			answerStatus = ai.AnswerStatusUnknown
		}

		out = append(out, normalizedDraftQuestion{
			Text:         text,
			Type:         qType,
			AnswerStatus: answerStatus,
			Explanation:  strings.TrimSpace(q.Explanation),
			CorrectIndex: correctIndex,
			Options:      opts,
		})
	}
	return out, nil
}

func deleteQuizQuestions(ctx context.Context, tx pgx.Tx, quizID int64) error {
	// Questions are per-draft; deleting them cascades to options, explanations,
	// and quiz_questions via ON DELETE CASCADE.
	_, err := tx.Exec(ctx, `
		DELETE FROM questions
		WHERE id IN (SELECT question_id FROM quiz_questions WHERE quiz_id = $1)
	`, quizID)
	return err
}

func insertDraftQuestion(ctx context.Context, tx pgx.Tx, quizID int64, chapterID *int64, contentHash string, q normalizedDraftQuestion, orderNo int) error {
	var questionID int64
	var hashArg any
	if contentHash != "" {
		hashArg = contentHash
	}
	err := tx.QueryRow(ctx, `
		INSERT INTO questions (chapter_id, content_hash, question_type, question_text, difficulty, source_type, status, answer_status, created_at)
		VALUES ($1, $2, $3, $4, 'medium', 'scanned_existing', 'active', $5, now())
		RETURNING id
	`, chapterID, hashArg, q.Type, q.Text, q.AnswerStatus).Scan(&questionID)
	if err != nil {
		return err
	}
	for idx, o := range q.Options {
		isCorrect := idx == q.CorrectIndex
		if _, err := tx.Exec(ctx, `
			INSERT INTO question_options (question_id, option_label, option_text, is_correct, created_at)
			VALUES ($1, $2, $3, $4, now())
		`, questionID, o.Label, o.Text, isCorrect); err != nil {
			return err
		}
	}
	if strings.TrimSpace(q.Explanation) != "" {
		if _, err := tx.Exec(ctx, `
			INSERT INTO question_explanations (question_id, style, explanation_text, language, created_at)
			VALUES ($1, 'simple', $2, $3, now())
		`, questionID, q.Explanation, ai.ExplanationDBCode(ai.DetectContentLanguage(q.Text))); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO quiz_questions (quiz_id, question_id, order_no, created_at)
		VALUES ($1, $2, $3, now())
	`, quizID, questionID, orderNo)
	return err
}
