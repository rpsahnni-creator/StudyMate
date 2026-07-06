package quiz

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Common service errors mapped to HTTP status codes by the handler.
var (
	ErrQuizNotFound     = errors.New("quiz not found")
	ErrAttemptNotFound  = errors.New("attempt not found")
	ErrAttemptForbidden = errors.New("attempt does not belong to user")
	ErrAttemptNotDone   = errors.New("attempt is not completed")
)

// Service holds quiz business logic backed by Postgres.
type Service struct {
	db     *pgxpool.Pool
	cache  *redis.Client
	logger *slog.Logger
}

func NewService(db *pgxpool.Pool, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{db: db, logger: logger}
}

// WithCache attaches a Valkey/Redis client used to cache analytics responses.
func (s *Service) WithCache(cache *redis.Client) *Service {
	s.cache = cache
	return s
}

// GetQuiz returns the quiz with its questions and options, without answers.
func (s *Service) GetQuiz(ctx context.Context, quizID, userID int64) (*QuizDetail, error) {
	var detail QuizDetail
	var subject, board *string
	err := s.db.QueryRow(ctx, `
		SELECT q.id, q.title, q.total_questions, b.subject, b.board
		FROM quizzes q
		LEFT JOIN chapters c ON c.id = q.chapter_id
		LEFT JOIN books b ON b.id = c.book_id
		WHERE q.id = $1
	`, quizID).Scan(&detail.ID, &detail.Title, &detail.TotalQuestions, &subject, &board)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrQuizNotFound
	}
	if err != nil {
		return nil, err
	}
	detail.Subject = deref(subject)
	detail.Board = deref(board)

	rows, err := s.db.Query(ctx, `
		SELECT ques.id, ques.question_text, ques.question_type
		FROM quiz_questions qq
		JOIN questions ques ON ques.id = qq.question_id
		WHERE qq.quiz_id = $1
		ORDER BY qq.order_no ASC
	`, quizID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	questionIndex := make(map[int64]int)
	for rows.Next() {
		var q QuizDetailQuestion
		if err := rows.Scan(&q.ID, &q.Text, &q.Type); err != nil {
			return nil, err
		}
		q.Options = []QuizDetailOption{}
		questionIndex[q.ID] = len(detail.Questions)
		detail.Questions = append(detail.Questions, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(detail.Questions) > 0 {
		optRows, err := s.db.Query(ctx, `
			SELECT o.question_id, o.id, o.option_label, o.option_text
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
			var opt QuizDetailOption
			if err := optRows.Scan(&questionID, &opt.ID, &opt.Label, &opt.Text); err != nil {
				return nil, err
			}
			if idx, ok := questionIndex[questionID]; ok {
				detail.Questions[idx].Options = append(detail.Questions[idx].Options, opt)
			}
		}
		if err := optRows.Err(); err != nil {
			return nil, err
		}
	}

	if detail.TotalQuestions == 0 {
		detail.TotalQuestions = len(detail.Questions)
	}
	detail.TimeLimit = timeLimitForQuestions(detail.TotalQuestions)
	return &detail, nil
}

// CreateAttempt returns the existing in-progress attempt or creates a new one.
func (s *Service) CreateAttempt(ctx context.Context, quizID, userID int64, startedAt time.Time) (*Attempt, error) {
	totalQuestions, err := s.quizQuestionCount(ctx, quizID)
	if err != nil {
		return nil, err
	}
	timeLimit := timeLimitForQuestions(totalQuestions)

	var existing Attempt
	err = s.db.QueryRow(ctx, `
		SELECT id, quiz_id, user_id, status, started_at
		FROM quiz_attempts
		WHERE quiz_id = $1 AND user_id = $2 AND status = $3
		ORDER BY started_at DESC
		LIMIT 1
	`, quizID, userID, AttemptStatusInProgress).Scan(
		&existing.ID, &existing.QuizID, &existing.UserID, &existing.Status, &existing.StartedAt,
	)
	if err == nil {
		existing.ExpiresAt = existing.StartedAt.Add(time.Duration(timeLimit) * time.Second)
		return &existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}

	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}
	var attempt Attempt
	err = s.db.QueryRow(ctx, `
		INSERT INTO quiz_attempts (quiz_id, user_id, started_at, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, quiz_id, user_id, status, started_at
	`, quizID, userID, startedAt, AttemptStatusInProgress).Scan(
		&attempt.ID, &attempt.QuizID, &attempt.UserID, &attempt.Status, &attempt.StartedAt,
	)
	if err != nil {
		return nil, err
	}
	attempt.ExpiresAt = attempt.StartedAt.Add(time.Duration(timeLimit) * time.Second)
	return &attempt, nil
}

// SubmitAttempt scores the attempt within a transaction. It is idempotent.
func (s *Service) SubmitAttempt(ctx context.Context, attemptID, userID int64, answers []Answer, submittedAt time.Time) (*AttemptResult, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var (
		quizID      int64
		ownerID     int64
		status      string
		startedAt   time.Time
		submittedDB *time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT quiz_id, user_id, status, started_at, submitted_at
		FROM quiz_attempts
		WHERE id = $1
		FOR UPDATE
	`, attemptID).Scan(&quizID, &ownerID, &status, &startedAt, &submittedDB)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAttemptNotFound
	}
	if err != nil {
		return nil, err
	}
	if ownerID != userID {
		return nil, ErrAttemptForbidden
	}

	// Idempotent: already submitted → return existing result.
	if status != AttemptStatusInProgress {
		result, err := s.loadResult(ctx, tx, attemptID)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}
		return result, nil
	}

	// Build correct-option map for all questions in this quiz.
	correctByQuestion, orderedQuestions, err := quizAnswerKey(ctx, tx, quizID)
	if err != nil {
		return nil, err
	}

	selectedByQuestion := make(map[int64]*int64, len(answers))
	for _, a := range answers {
		selectedByQuestion[a.QuestionID] = a.SelectedOptionID
	}

	total := len(orderedQuestions)
	correct, wrong, skipped := 0, 0, 0
	answerRows := make([]attemptAnswerRow, 0, total)
	answeredAt := time.Now().UTC()

	for _, questionID := range orderedQuestions {
		selected, provided := selectedByQuestion[questionID]
		correctOption := correctByQuestion[questionID]

		var isCorrect *bool
		switch {
		case !provided || selected == nil:
			skipped++
		case correctOption != nil && *selected == *correctOption:
			correct++
			t := true
			isCorrect = &t
		default:
			wrong++
			f := false
			isCorrect = &f
		}

		answerRows = append(answerRows, attemptAnswerRow{
			questionID: questionID,
			selected:   selected,
			isCorrect:  isCorrect,
		})
	}

	if err := insertAttemptAnswers(ctx, tx, attemptID, answerRows, answeredAt); err != nil {
		return nil, err
	}

	score := scorePercent(correct, total)
	if submittedAt.IsZero() {
		submittedAt = time.Now().UTC()
	}
	timeTaken := int(submittedAt.Sub(startedAt).Seconds())
	if timeTaken < 0 {
		timeTaken = 0
	}

	if _, err := tx.Exec(ctx, `
		UPDATE quiz_attempts
		SET status = $1, score = $2, correct_count = $3, wrong_count = $4, skipped_count = $5, submitted_at = $6
		WHERE id = $7
	`, AttemptStatusCompleted, score, correct, wrong, skipped, submittedAt, attemptID); err != nil {
		return nil, err
	}

	summary := fmt.Sprintf("Scored %.1f%% (%d correct, %d wrong, %d skipped of %d).", score, correct, wrong, skipped, total)
	if _, err := tx.Exec(ctx, `
		INSERT INTO quiz_reports (attempt_id, accuracy, weak_topics_json, summary_text, created_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (attempt_id) DO NOTHING
	`, attemptID, score, "[]", summary); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Don't block the submit response on cache invalidation.
	go s.invalidateAnalytics(context.Background(), userID)

	return &AttemptResult{
		AttemptID:      attemptID,
		Score:          score,
		CorrectCount:   correct,
		WrongCount:     wrong,
		SkippedCount:   skipped,
		TotalQuestions: total,
		TimeTaken:      timeTaken,
	}, nil
}

// GetReview returns per-question correctness for a completed attempt.
func (s *Service) GetReview(ctx context.Context, attemptID, userID int64) (*ReviewDetail, error) {
	var (
		quizID  int64
		ownerID int64
		status  string
		score   *float64
	)
	err := s.db.QueryRow(ctx, `
		SELECT quiz_id, user_id, status, score
		FROM quiz_attempts WHERE id = $1
	`, attemptID).Scan(&quizID, &ownerID, &status, &score)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrAttemptNotFound
	}
	if err != nil {
		return nil, err
	}
	if ownerID != userID {
		return nil, ErrAttemptForbidden
	}
	if status != AttemptStatusCompleted {
		return nil, ErrAttemptNotDone
	}

	review := &ReviewDetail{Score: derefFloat(score), Questions: []ReviewQuestion{}}

	rows, err := s.db.Query(ctx, `
		SELECT ques.id, ques.question_text,
		       (SELECT o.id FROM question_options o WHERE o.question_id = ques.id AND o.is_correct = true LIMIT 1) AS correct_option_id,
		       (SELECT o.option_text FROM question_options o WHERE o.question_id = ques.id AND o.is_correct = true LIMIT 1) AS correct_option_text,
		       aa.selected_option_id,
		       (SELECT o.option_text FROM question_options o WHERE o.id = aa.selected_option_id) AS selected_option_text,
		       aa.is_correct,
		       COALESCE(
		         (SELECT e.explanation_text FROM question_explanations e WHERE e.question_id = ques.id ORDER BY e.id ASC LIMIT 1),
		         ''
		       ) AS explanation
		FROM quiz_questions qq
		JOIN questions ques ON ques.id = qq.question_id
		LEFT JOIN quiz_attempt_answers aa ON aa.question_id = ques.id AND aa.attempt_id = $2
		WHERE qq.quiz_id = $1
		ORDER BY qq.order_no ASC
	`, quizID, attemptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			rq                 ReviewQuestion
			correctOpt         *int64
			correctText        *string
			selectedOpt        *int64
			selectedText       *string
			isCorrect          *bool
		)
		if err := rows.Scan(&rq.ID, &rq.Text, &correctOpt, &correctText, &selectedOpt, &selectedText, &isCorrect, &rq.Explanation); err != nil {
			return nil, err
		}
		rq.CorrectAnswer = correctOpt
		rq.YourAnswer = selectedOpt
		if correctText != nil {
			rq.CorrectAnswerText = *correctText
		}
		if selectedText != nil {
			rq.YourAnswerText = *selectedText
		}
		switch {
		case selectedOpt == nil:
			rq.Status = answerStatusSkipped
			rq.IsCorrect = false
		case isCorrect != nil && *isCorrect:
			rq.Status = answerStatusCorrect
			rq.IsCorrect = true
		default:
			rq.Status = answerStatusWrong
			rq.IsCorrect = false
		}
		review.Questions = append(review.Questions, rq)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return review, nil
}

// GetUserReports returns a paginated list of completed attempts, latest first.
func (s *Service) GetUserReports(ctx context.Context, userID int64, page, limit int) (*ReportsPage, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}
	offset := (page - 1) * limit

	var total int
	if err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM quiz_attempts WHERE user_id = $1 AND status = $2
	`, userID, AttemptStatusCompleted).Scan(&total); err != nil {
		return nil, err
	}

	rows, err := s.db.Query(ctx, `
		SELECT a.id, a.quiz_id, q.title, COALESCE(a.score, 0), a.submitted_at
		FROM quiz_attempts a
		JOIN quizzes q ON q.id = a.quiz_id
		WHERE a.user_id = $1 AND a.status = $2
		ORDER BY a.submitted_at DESC NULLS LAST
		LIMIT $3 OFFSET $4
	`, userID, AttemptStatusCompleted, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := &ReportsPage{Reports: []ReportItem{}, Total: total, Page: page}
	for rows.Next() {
		var item ReportItem
		var submittedAt *time.Time
		if err := rows.Scan(&item.AttemptID, &item.QuizID, &item.QuizTitle, &item.Score, &submittedAt); err != nil {
			return nil, err
		}
		if submittedAt != nil {
			item.CompletedAt = *submittedAt
		}
		result.Reports = append(result.Reports, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *Service) quizQuestionCount(ctx context.Context, quizID int64) (int, error) {
	var total int
	err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM quiz_questions WHERE quiz_id = $1`, quizID).Scan(&total)
	if err != nil {
		return 0, err
	}
	if total == 0 {
		// Fall back to the stored total_questions; also verifies the quiz exists.
		err = s.db.QueryRow(ctx, `SELECT total_questions FROM quizzes WHERE id = $1`, quizID).Scan(&total)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrQuizNotFound
		}
		if err != nil {
			return 0, err
		}
	}
	return total, nil
}

func (s *Service) loadResult(ctx context.Context, tx pgx.Tx, attemptID int64) (*AttemptResult, error) {
	var (
		score        *float64
		correct      int
		wrong        int
		skipped      int
		startedAt    time.Time
		submittedAt  *time.Time
	)
	err := tx.QueryRow(ctx, `
		SELECT score, correct_count, wrong_count, skipped_count, started_at, submitted_at
		FROM quiz_attempts WHERE id = $1
	`, attemptID).Scan(&score, &correct, &wrong, &skipped, &startedAt, &submittedAt)
	if err != nil {
		return nil, err
	}
	timeTaken := 0
	if submittedAt != nil {
		timeTaken = int(submittedAt.Sub(startedAt).Seconds())
		if timeTaken < 0 {
			timeTaken = 0
		}
	}
	return &AttemptResult{
		AttemptID:      attemptID,
		Score:          derefFloat(score),
		CorrectCount:   correct,
		WrongCount:     wrong,
		SkippedCount:   skipped,
		TotalQuestions: correct + wrong + skipped,
		TimeTaken:      timeTaken,
	}, nil
}

// quizAnswerKey returns the correct option per question and the ordered question IDs.
func quizAnswerKey(ctx context.Context, tx pgx.Tx, quizID int64) (map[int64]*int64, []int64, error) {
	rows, err := tx.Query(ctx, `
		SELECT ques.id, correct_opt.id
		FROM quiz_questions qq
		JOIN questions ques ON ques.id = qq.question_id
		LEFT JOIN LATERAL (
			SELECT o.id
			FROM question_options o
			WHERE o.question_id = ques.id AND o.is_correct = true
			LIMIT 1
		) correct_opt ON true
		WHERE qq.quiz_id = $1
		ORDER BY qq.order_no ASC
	`, quizID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	correctByQuestion := make(map[int64]*int64)
	var ordered []int64
	for rows.Next() {
		var questionID int64
		var correctOpt *int64
		if err := rows.Scan(&questionID, &correctOpt); err != nil {
			return nil, nil, err
		}
		correctByQuestion[questionID] = correctOpt
		ordered = append(ordered, questionID)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	return correctByQuestion, ordered, nil
}

type attemptAnswerRow struct {
	questionID int64
	selected   *int64
	isCorrect  *bool
}

func insertAttemptAnswers(ctx context.Context, tx pgx.Tx, attemptID int64, rows []attemptAnswerRow, answeredAt time.Time) error {
	if len(rows) == 0 {
		return nil
	}
	_, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"quiz_attempt_answers"},
		[]string{"attempt_id", "question_id", "selected_option_id", "is_correct", "answered_at"},
		pgx.CopyFromSlice(len(rows), func(i int) ([]any, error) {
			row := rows[i]
			var selected any
			if row.selected != nil {
				selected = *row.selected
			}
			return []any{attemptID, row.questionID, selected, row.isCorrect, answeredAt}, nil
		}),
	)
	return err
}

func scorePercent(correct, total int) float64 {
	if total <= 0 {
		return 0
	}
	raw := (float64(correct) / float64(total)) * 100
	return math.Round(raw*10) / 10
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}
