import { apiCall } from "./auth";

// --- Shared quiz types (mirror backend/internal/quiz DTOs) ---

export interface QuizOption {
  id: number;
  label: string;
  text: string;
}

export interface QuizQuestion {
  id: number;
  text: string;
  type: string;
  options: QuizOption[];
}

export interface QuizDetail {
  id: number;
  title: string;
  subject: string;
  board: string;
  totalQuestions: number;
  timeLimit: number;
  questions: QuizQuestion[];
}

export interface StartAttemptResponse {
  attemptId: number;
  expiresAt: string;
}

export interface AnswerInput {
  questionId: number;
  selectedOptionId: number | null;
}

export interface AttemptResult {
  attemptId: number;
  score: number;
  correctCount: number;
  wrongCount: number;
  skippedCount: number;
  totalQuestions: number;
  timeTaken: number;
}

export interface ReviewQuestion {
  id: number;
  text: string;
  yourAnswer: number | null;
  yourAnswerText?: string;
  correctAnswer: number | null;
  correctAnswerText?: string;
  isCorrect: boolean;
  status: "correct" | "wrong" | "skipped";
  explanation: string;
}

export interface ReviewDetail {
  score: number;
  questions: ReviewQuestion[];
}

export interface ReportItem {
  attemptId: number;
  quizId: number;
  quizTitle: string;
  score: number;
  completedAt: string;
}

export interface ReportsPage {
  reports: ReportItem[];
  total: number;
  page: number;
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await apiCall(path, options);
  if (!response.ok) {
    let message = `Request failed (${response.status})`;
    try {
      const data = (await response.json()) as { error?: string; message?: string };
      message = data.error ?? data.message ?? message;
    } catch {
      // keep default message
    }
    throw new Error(message);
  }
  return (await response.json()) as T;
}

export function getQuiz(quizId: string): Promise<QuizDetail> {
  return request<QuizDetail>(`/quizzes/${quizId}`);
}

export function startAttempt(quizId: string): Promise<StartAttemptResponse> {
  return request<StartAttemptResponse>(`/quizzes/${quizId}/attempts`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ startedAt: new Date().toISOString() }),
  });
}

export function submitAttempt(
  quizId: string,
  attemptId: string,
  answers: AnswerInput[]
): Promise<AttemptResult> {
  return request<AttemptResult>(`/quizzes/${quizId}/attempts/${attemptId}/submit`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ answers, submittedAt: new Date().toISOString() }),
  });
}

export function getReview(quizId: string, attemptId: string): Promise<ReviewDetail> {
  return request<ReviewDetail>(`/quizzes/${quizId}/attempts/${attemptId}/review`);
}

export function getMyReports(page = 1, limit = 10): Promise<ReportsPage> {
  return request<ReportsPage>(`/users/me/reports?page=${page}&limit=${limit}`);
}

// --- Question-scan draft review / publish ---

export interface DraftOption {
  id: number;
  label: string;
  text: string;
  isCorrect: boolean;
}

export interface DraftQuestion {
  id: number;
  text: string;
  type: string; // mcq | fill_blank | true_false
  answerStatus: "set" | "unknown";
  explanation: string;
  options: DraftOption[];
}

export interface DraftDetail {
  id: number;
  title: string;
  status: "draft" | "published";
  totalQuestions: number;
  needsAnswerCount: number;
  questions: DraftQuestion[];
}

export interface DraftOptionInput {
  label: string;
  text: string;
}

export interface DraftQuestionInput {
  text: string;
  type: string;
  correctIndex: number; // -1 = answer not chosen yet
  explanation: string;
  options: DraftOptionInput[];
}

export interface SaveDraftRequest {
  questions: DraftQuestionInput[];
}

export interface PublishResult {
  quizId: number;
  status: string;
  totalQuestions: number;
}

export function getDraft(quizId: string): Promise<DraftDetail> {
  return request<DraftDetail>(`/quizzes/${quizId}/draft`);
}

export function saveDraft(quizId: string, body: SaveDraftRequest): Promise<DraftDetail> {
  return request<DraftDetail>(`/quizzes/${quizId}/draft`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
}

export function publishDraft(quizId: string): Promise<PublishResult> {
  return request<PublishResult>(`/quizzes/${quizId}/publish`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({}),
  });
}

// --- Analytics types (mirror backend/internal/quiz analytics DTOs) ---

export interface AnalyticsSummary {
  totalQuizzes: number;
  averageScore: number;
  totalQuestionsAttempted: number;
  correctAnswers: number;
  studyStreakDays: number;
  thisWeekScore: number;
  lastWeekScore: number;
  improvement: number;
}

export interface SubjectBreakdown {
  subject: string;
  quizCount: number;
  avgScore: number;
  trend: "improving" | "declining" | "stable";
}

export interface WeeklyScore {
  week: string;
  score: number;
  quizCount: number;
}

export interface DailyActivity {
  date: string;
  quizCount: number;
  avgScore: number;
}

export interface Analytics {
  summary: AnalyticsSummary;
  subjectBreakdown: SubjectBreakdown[];
  weeklyScores: WeeklyScore[];
  recentActivity: DailyActivity[];
}

export interface TopicAccuracy {
  topic: string;
  subject: string;
  accuracy: number;
  totalAnswered: number;
  correctCount: number;
  sampleQuizId: number | null;
}

export interface TopicAnalytics {
  topics: TopicAccuracy[];
}

export function getAnalytics(): Promise<Analytics> {
  return request<Analytics>(`/users/me/analytics`);
}

export function getTopicAnalytics(): Promise<TopicAnalytics> {
  return request<TopicAnalytics>(`/users/me/analytics/topics`);
}
