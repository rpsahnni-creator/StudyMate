package quiz

import (
	"errors"
	"time"
)

// defaultTimeLimitPerQuestion is used to derive a quiz time limit because the
// quizzes table has no explicit time_limit column in Phase 1.
const defaultTimeLimitPerQuestion = 60 // seconds

// QuizDetail is the public quiz payload returned before an attempt. It never
// includes the correct answer for any option.
type QuizDetail struct {
	ID             int64                `json:"id"`
	Title          string               `json:"title"`
	Subject        string               `json:"subject"`
	Board          string               `json:"board"`
	TotalQuestions int                  `json:"totalQuestions"`
	TimeLimit      int                  `json:"timeLimit"`
	Questions      []QuizDetailQuestion `json:"questions"`
}

type QuizDetailQuestion struct {
	ID      int64              `json:"id"`
	Text    string             `json:"text"`
	Type    string             `json:"type"`
	Options []QuizDetailOption `json:"options"`
}

type QuizDetailOption struct {
	ID    int64  `json:"id"`
	Label string `json:"label"`
	Text  string `json:"text"`
}

// CreateAttemptRequest starts an attempt.
type CreateAttemptRequest struct {
	StartedAt time.Time `json:"startedAt"`
}

func (r *CreateAttemptRequest) Validate() error {
	if r.StartedAt.IsZero() {
		r.StartedAt = time.Now().UTC()
	}
	return nil
}

type CreateAttemptResponse struct {
	AttemptID int64     `json:"attemptId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Attempt is the domain representation of a quiz attempt.
type Attempt struct {
	ID          int64      `json:"id"`
	QuizID      int64      `json:"quizId"`
	UserID      int64      `json:"userId"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"startedAt"`
	SubmittedAt *time.Time `json:"submittedAt,omitempty"`
	ExpiresAt   time.Time  `json:"expiresAt"`
}

// Answer is a single submitted answer. SelectedOptionID is nil when skipped.
type Answer struct {
	QuestionID       int64  `json:"questionId"`
	SelectedOptionID *int64 `json:"selectedOptionId"`
}

// SubmitAttemptRequest carries the user's answers.
type SubmitAttemptRequest struct {
	Answers     []Answer  `json:"answers"`
	SubmittedAt time.Time `json:"submittedAt"`
}

func (r *SubmitAttemptRequest) Validate() error {
	if r.Answers == nil {
		return errors.New("answers array is required")
	}
	seen := make(map[int64]struct{}, len(r.Answers))
	for _, a := range r.Answers {
		if a.QuestionID <= 0 {
			return errors.New("each answer must include a valid questionId")
		}
		if _, dup := seen[a.QuestionID]; dup {
			return errors.New("duplicate questionId in answers")
		}
		seen[a.QuestionID] = struct{}{}
	}
	if r.SubmittedAt.IsZero() {
		r.SubmittedAt = time.Now().UTC()
	}
	return nil
}

// AttemptResult is returned after a successful (or idempotent) submit.
type AttemptResult struct {
	AttemptID      int64   `json:"attemptId"`
	Score          float64 `json:"score"`
	CorrectCount   int     `json:"correctCount"`
	WrongCount     int     `json:"wrongCount"`
	SkippedCount   int     `json:"skippedCount"`
	TotalQuestions int     `json:"totalQuestions"`
	TimeTaken      int     `json:"timeTaken"`
}

// ReviewDetail is returned only for completed attempts and includes answers.
type ReviewDetail struct {
	Score     float64          `json:"score"`
	Questions []ReviewQuestion `json:"questions"`
}

type ReviewQuestion struct {
	ID                 int64  `json:"id"`
	Text               string `json:"text"`
	YourAnswer         *int64 `json:"yourAnswer"`
	YourAnswerText     string `json:"yourAnswerText,omitempty"`
	CorrectAnswer      *int64 `json:"correctAnswer"`
	CorrectAnswerText  string `json:"correctAnswerText,omitempty"`
	IsCorrect          bool   `json:"isCorrect"`
	Status             string `json:"status"` // correct | wrong | skipped
	Explanation        string `json:"explanation"`
}

// ReportsPage is a paginated list of the user's completed attempts.
type ReportsPage struct {
	Reports []ReportItem `json:"reports"`
	Total   int          `json:"total"`
	Page    int          `json:"page"`
}

type ReportItem struct {
	AttemptID   int64     `json:"attemptId"`
	QuizID      int64     `json:"quizId"`
	QuizTitle   string    `json:"quizTitle"`
	Score       float64   `json:"score"`
	CompletedAt time.Time `json:"completedAt"`
}

// --- Analytics (GET /users/me/analytics) ---

// Analytics is the full analytics dashboard payload.
type Analytics struct {
	Summary          AnalyticsSummary   `json:"summary"`
	SubjectBreakdown []SubjectBreakdown `json:"subjectBreakdown"`
	WeeklyScores     []WeeklyScore      `json:"weeklyScores"`
	RecentActivity   []DailyActivity    `json:"recentActivity"`
}

type AnalyticsSummary struct {
	TotalQuizzes            int     `json:"totalQuizzes"`
	AverageScore            float64 `json:"averageScore"`
	TotalQuestionsAttempted int     `json:"totalQuestionsAttempted"`
	CorrectAnswers          int     `json:"correctAnswers"`
	StudyStreakDays         int     `json:"studyStreakDays"`
	ThisWeekScore           float64 `json:"thisWeekScore"`
	LastWeekScore           float64 `json:"lastWeekScore"`
	Improvement             float64 `json:"improvement"`
}

type SubjectBreakdown struct {
	Subject   string  `json:"subject"`
	QuizCount int     `json:"quizCount"`
	AvgScore  float64 `json:"avgScore"`
	Trend     string  `json:"trend"` // improving | declining | stable
}

type WeeklyScore struct {
	Week      string  `json:"week"` // e.g. "2025-W01"
	Score     float64 `json:"score"`
	QuizCount int     `json:"quizCount"`
}

type DailyActivity struct {
	Date      string  `json:"date"` // YYYY-MM-DD
	QuizCount int     `json:"quizCount"`
	AvgScore  float64 `json:"avgScore"`
}

// TopicAnalytics is the topic-level breakdown (GET /users/me/analytics/topics).
type TopicAnalytics struct {
	Topics []TopicAccuracy `json:"topics"`
}

type TopicAccuracy struct {
	Topic         string  `json:"topic"`
	Subject       string  `json:"subject"`
	Accuracy      float64 `json:"accuracy"` // 0-100
	TotalAnswered int     `json:"totalAnswered"`
	CorrectCount  int     `json:"correctCount"`
	SampleQuizID  *int64  `json:"sampleQuizId"` // a quiz covering this topic, for "Practice this"
}

const (
	AttemptStatusInProgress = "in_progress"
	AttemptStatusCompleted  = "completed"

	answerStatusCorrect = "correct"
	answerStatusWrong   = "wrong"
	answerStatusSkipped = "skipped"
)

func timeLimitForQuestions(total int) int {
	if total <= 0 {
		return defaultTimeLimitPerQuestion
	}
	return total * defaultTimeLimitPerQuestion
}
