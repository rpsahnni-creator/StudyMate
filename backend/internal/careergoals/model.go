package careergoals

import (
	"errors"
	"time"
)

// Service/handler errors mapped to HTTP status codes.
var (
	ErrNoActiveGoal = errors.New("no active goal")
	ErrGoalNotFound = errors.New("career goal not found")
	ErrSetNotFound  = errors.New("practice set not found")
	ErrSetForbidden = errors.New("practice set does not belong to user")
	ErrInvalidInput = errors.New("invalid input")
)

// dateLayout is the wire format used for all date-only fields.
const dateLayout = "2006-01-02"

// CareerGoal is a selectable goal in the discovery catalog (GET /goals).
type CareerGoal struct {
	ID           int64    `json:"id"`
	Slug         string   `json:"slug"`
	Name         string   `json:"name"` // maps to career_goals.title
	Description  string   `json:"description"`
	ExamName     string   `json:"examName"`
	TargetMonths int      `json:"targetMonths"`
	SubjectAreas []string `json:"subjectAreas"`
}

// SelectGoalRequest is the body for POST /goals/select.
type SelectGoalRequest struct {
	GoalID     int64  `json:"goalId"`
	TargetDate string `json:"targetDate"` // YYYY-MM-DD, optional
}

// SelectGoalResponse is returned after selecting/switching a goal.
type SelectGoalResponse struct {
	StudentGoalID int64 `json:"studentGoalId"`
}

// GoalProgress summarizes the student's activity against their active goal.
type GoalProgress struct {
	TotalPractices     int      `json:"totalPractices"`
	CompletedPractices int      `json:"completedPractices"`
	CurrentStreak      int      `json:"currentStreak"`
	TodayStatus        string   `json:"todayStatus"` // "none" | "pending" | "completed"
	TodayScore         *float64 `json:"todayScore"`
	AverageScore       *float64 `json:"averageScore"`
}

// MyGoal is the current active goal with a progress summary (GET /goals/my).
type MyGoal struct {
	StudentGoalID int64        `json:"studentGoalId"`
	GoalID        int64        `json:"goalId"`
	Name          string       `json:"name"`
	ExamName      string       `json:"examName"`
	TargetDate    *string      `json:"targetDate"`
	Status        string       `json:"status"`
	DaysRemaining *int         `json:"daysRemaining"`
	SubjectAreas  []string     `json:"subjectAreas"`
	Progress      GoalProgress `json:"progress"`
}

// PracticeOption / PracticeQuestion mirror the quiz UI's question shape.
type PracticeOption struct {
	ID    int64  `json:"id"`
	Label string `json:"label"`
	Text  string `json:"text"`
}

type PracticeQuestion struct {
	ID      int64            `json:"id"`
	Text    string           `json:"text"`
	Type    string           `json:"type"`
	Topic   string           `json:"topic"`
	Subject string           `json:"subject"`
	Options []PracticeOption `json:"options"`
}

// TodayPractice is the daily set returned by GET /goals/my/practice/today.
type TodayPractice struct {
	SetID      int64              `json:"setId"`
	Date       string             `json:"date"`
	Status     string             `json:"status"`
	Score      *float64           `json:"score"`
	TopicFocus []string           `json:"topicFocus"`
	Questions  []PracticeQuestion `json:"questions"`
}

// PracticeAnswer is a single submitted answer.
type PracticeAnswer struct {
	QuestionID       int64  `json:"questionId"`
	SelectedOptionID *int64 `json:"selectedOptionId"`
}

// SubmitPracticeRequest is the body for POST /goals/my/practice/{setId}/submit.
type SubmitPracticeRequest struct {
	Answers     []PracticeAnswer `json:"answers"`
	SubmittedAt time.Time        `json:"submittedAt"`
}

// SkillDelta describes how a topic moved after a practice submission.
type SkillDelta struct {
	Topic     string  `json:"topic"`
	Subject   string  `json:"subject"`
	Direction string  `json:"direction"` // "improved" | "needs_work" | "unchanged"
	Score     float64 `json:"score"`     // this set's score for the topic (0-100)
}

// SubmitPracticeResult is returned after scoring a daily set.
type SubmitPracticeResult struct {
	SetID          int64        `json:"setId"`
	Score          float64      `json:"score"`
	CorrectCount   int          `json:"correctCount"`
	WrongCount     int          `json:"wrongCount"`
	SkippedCount   int          `json:"skippedCount"`
	TotalQuestions int          `json:"totalQuestions"`
	SkillUpdates   []SkillDelta `json:"skillUpdates"`
}

// PracticeHistoryItem is one day's set for the trend chart.
type PracticeHistoryItem struct {
	SetID  int64    `json:"setId"`
	Date   string   `json:"date"`
	Status string   `json:"status"`
	Score  *float64 `json:"score"`
}

// PracticeHistory is the last-30-days response.
type PracticeHistory struct {
	Items []PracticeHistoryItem `json:"items"`
}

// SkillGap is one topic's weakness row (GET /goals/my/skills).
type SkillGap struct {
	Subject       string  `json:"subject"`
	Topic         string  `json:"topic"`
	WeaknessScore float64 `json:"weaknessScore"`
	LastPracticed *string `json:"lastPracticed"`
}
