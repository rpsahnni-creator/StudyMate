import { API_URL, fetchWithAuth } from "./auth";

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

// GET /goals is a public catalog endpoint — no auth required.
export async function getGoals(): Promise<CareerGoal[]> {
  const res = await fetch(`${API_URL}/goals`, { cache: "no-store" });
  if (!res.ok) {
    throw new Error("Failed to load career goals");
  }
  return (await res.json()) as CareerGoal[];
}

// Returns null when the user has no active goal yet (backend returns 404),
// instead of throwing — callers can treat that as "show goal picker".
export async function getMyGoal(): Promise<MyGoal | null> {
  const res = await fetchWithAuth(`${API_URL}/goals/my`);
  if (res.status === 404) return null;
  if (!res.ok) {
    throw new Error("Failed to load your goal");
  }
  return (await res.json()) as MyGoal;
}

export async function selectGoal(goalId: number): Promise<{ studentGoalId: number }> {
  const res = await fetchWithAuth(`${API_URL}/goals/select`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ goalId }),
  });
  if (!res.ok) {
    let message = "Failed to select goal";
    try {
      const data = (await res.json()) as { message?: string; error?: string };
      message = data.message ?? data.error ?? message;
    } catch {
      // keep default message
    }
    throw new Error(message);
  }
  return (await res.json()) as { studentGoalId: number };
}

export async function abandonGoal(): Promise<void> {
  const res = await fetchWithAuth(`${API_URL}/goals/my`, { method: "DELETE" });
  if (!res.ok) {
    throw new Error("Failed to update goal");
  }
}
