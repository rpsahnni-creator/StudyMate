import { apiCall } from "./auth";

// --- Types (mirror backend/internal/careergoals DTOs) ---

export interface CareerGoal {
  id: number;
  slug: string;
  name: string;
  description: string;
  examName: string;
  targetMonths: number;
  subjectAreas: string[];
}

export interface GoalProgress {
  totalPractices: number;
  completedPractices: number;
  currentStreak: number;
  todayStatus: "none" | "pending" | "completed";
  todayScore: number | null;
  averageScore: number | null;
}

export interface MyGoal {
  studentGoalId: number;
  goalId: number;
  name: string;
  examName: string;
  targetDate: string | null;
  status: string;
  daysRemaining: number | null;
  subjectAreas: string[];
  progress: GoalProgress;
}

export interface PracticeOption {
  id: number;
  label: string;
  text: string;
}

export interface PracticeQuestion {
  id: number;
  text: string;
  type: string;
  topic: string;
  subject: string;
  options: PracticeOption[];
}

export interface TodayPractice {
  setId: number;
  date: string;
  status: "pending" | "completed";
  score: number | null;
  topicFocus: string[];
  questions: PracticeQuestion[];
}

export interface PracticeAnswerInput {
  questionId: number;
  selectedOptionId: number | null;
}

export interface SkillDelta {
  topic: string;
  subject: string;
  direction: "improved" | "needs_work" | "unchanged";
  score: number;
}

export interface SubmitPracticeResult {
  setId: number;
  score: number;
  correctCount: number;
  wrongCount: number;
  skippedCount: number;
  totalQuestions: number;
  skillUpdates: SkillDelta[];
}

export interface PracticeHistoryItem {
  setId: number;
  date: string;
  status: string;
  score: number | null;
}

export interface PracticeHistory {
  items: PracticeHistoryItem[];
}

export interface SkillGap {
  subject: string;
  topic: string;
  weaknessScore: number;
  lastPracticed: string | null;
}

// FeatureGatedError is thrown when the backend returns 403 (career_goals_module
// flag is OFF for this user). The UI treats this as "Coming Soon" rather than an
// error state.
export class FeatureGatedError extends Error {
  constructor() {
    super("feature_not_available");
    this.name = "FeatureGatedError";
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await apiCall(path, options);
  if (response.status === 403) {
    throw new FeatureGatedError();
  }
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
  // Some endpoints (e.g. DELETE) may return a tiny body; guard JSON parse.
  const text = await response.text();
  return (text ? JSON.parse(text) : {}) as T;
}

export function listGoals(): Promise<CareerGoal[]> {
  return request<CareerGoal[]>("/goals");
}

export function selectGoal(goalId: number, targetDate?: string): Promise<{ studentGoalId: number }> {
  return request<{ studentGoalId: number }>("/goals/select", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ goalId, targetDate: targetDate ?? "" }),
  });
}

export function getMyGoal(): Promise<MyGoal> {
  return request<MyGoal>("/goals/my");
}

export function abandonGoal(): Promise<{ status: string }> {
  return request<{ status: string }>("/goals/my", { method: "DELETE" });
}

export function getTodayPractice(): Promise<TodayPractice> {
  return request<TodayPractice>("/goals/my/practice/today");
}

export function submitPractice(
  setId: number,
  answers: PracticeAnswerInput[]
): Promise<SubmitPracticeResult> {
  return request<SubmitPracticeResult>(`/goals/my/practice/${setId}/submit`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ answers, submittedAt: new Date().toISOString() }),
  });
}

export function getPracticeHistory(): Promise<PracticeHistory> {
  return request<PracticeHistory>("/goals/my/practice/history");
}

export function getSkillGaps(): Promise<SkillGap[]> {
  return request<SkillGap[]>("/goals/my/skills");
}
