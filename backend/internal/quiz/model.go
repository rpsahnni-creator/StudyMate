package quiz

import "time"

type Question struct {
	ID            int64  `json:"id"`
	ChapterID     *int64 `json:"chapter_id,omitempty"`
	ContentHash   *string `json:"content_hash,omitempty"`
	QuestionType  string `json:"question_type"`
	QuestionText  string `json:"question_text"`
	Difficulty    *string `json:"difficulty,omitempty"`
	SourceType    string `json:"source_type"`
	Status        string `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

type QuestionOption struct {
	ID           int64  `json:"id"`
	QuestionID   int64  `json:"question_id"`
	OptionLabel  string `json:"option_label"`
	OptionText   string `json:"option_text"`
	IsCorrect    bool   `json:"is_correct"`
}

type QuestionExplanation struct {
	ID               int64  `json:"id"`
	QuestionID       int64  `json:"question_id"`
	Style            *string `json:"style,omitempty"`
	ExplanationText  string `json:"explanation_text"`
	Language         string `json:"language"`
}

type Quiz struct {
	ID              int64  `json:"id"`
	ChapterID       *int64 `json:"chapter_id,omitempty"`
	ContentHash     *string `json:"content_hash,omitempty"`
	Title           string `json:"title"`
	TotalQuestions  int    `json:"total_questions"`
	GenerationType  string `json:"generation_type"`
	CreatedAt       time.Time `json:"created_at"`
}

type QuizQuestion struct {
	ID          int64 `json:"id"`
	QuizID      int64 `json:"quiz_id"`
	QuestionID  int64 `json:"question_id"`
	OrderNo     int   `json:"order_no"`
}

type QuizResponse struct {
	Quiz       Quiz            `json:"quiz"`
	Questions  []Question      `json:"questions"`
	Options    map[int64][]QuestionOption `json:"options,omitempty"`
	Explanations map[int64]QuestionExplanation `json:"explanations,omitempty"`
}
