"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { getReview, type AttemptResult } from "../../../../../lib/api";

function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m <= 0) return `${s}s`;
  return `${m}m ${s}s`;
}

export default function QuizResultPage() {
  const params = useParams<{ quizId: string; attemptId: string }>();
  const { quizId, attemptId } = params;
  const router = useRouter();

  const [result, setResult] = useState<AttemptResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadResult = useCallback(async () => {
    setLoading(true);
    setError(null);
    // Prefer the result stored right after submit.
    try {
      const cached = localStorage.getItem(`studyapp.quiz.result.${attemptId}`);
      if (cached) {
        setResult(JSON.parse(cached) as AttemptResult);
        setLoading(false);
        return;
      }
    } catch {
      // ignore and fall back to review
    }
    // Fallback: derive counts from the review endpoint.
    try {
      const review = await getReview(quizId, attemptId);
      let correct = 0;
      let wrong = 0;
      let skipped = 0;
      for (const q of review.questions) {
        if (q.status === "correct") correct++;
        else if (q.status === "wrong") wrong++;
        else skipped++;
      }
      setResult({
        attemptId: Number(attemptId),
        score: review.score,
        correctCount: correct,
        wrongCount: wrong,
        skippedCount: skipped,
        totalQuestions: review.questions.length,
        timeTaken: 0,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load result");
    } finally {
      setLoading(false);
    }
  }, [quizId, attemptId]);

  useEffect(() => {
    void loadResult();
  }, [loadResult]);

  if (loading) {
    return (
      <main style={styles.container}>
        <p>Loading result…</p>
      </main>
    );
  }

  if (error || !result) {
    return (
      <main style={styles.container}>
        <p style={styles.error}>{error ?? "No result found"}</p>
        <button style={styles.primaryButton} onClick={() => void loadResult()}>
          Retry
        </button>
      </main>
    );
  }

  const passing = result.score >= 60;

  return (
    <main style={styles.container} className="animate-in">
      <div style={styles.card} className="card">
        <span style={styles.emoji}>{passing ? "🎉" : "💪"}</span>
        <h1 style={styles.title}>{passing ? "Well done!" : "Keep going!"}</h1>
        <div
          style={{
            ...styles.scoreCircle,
            borderColor: scoreColor(result.score),
            color: scoreColor(result.score),
            background: `${scoreColor(result.score)}14`,
          }}
        >
          {result.score.toFixed(1)}%
        </div>

        <div style={styles.statsRow}>
          <Stat label="Correct" value={result.correctCount} color="var(--success)" />
          <Stat label="Wrong" value={result.wrongCount} color="var(--danger)" />
          <Stat label="Skipped" value={result.skippedCount} color="var(--warning)" />
        </div>

        <p style={styles.meta}>
          {result.totalQuestions} questions · Time taken {formatDuration(result.timeTaken)}
        </p>

        <div style={styles.buttonRow}>
          <button
            style={styles.primaryButton}
            onClick={() => router.push(`/quiz/${quizId}/review/${attemptId}`)}
          >
            Review Answers
          </button>
          <button className="btn-reset" style={styles.secondaryButton} onClick={() => router.push("/")}>
            Dashboard
          </button>
        </div>
      </div>
    </main>
  );
}

function scoreColor(score: number): string {
  if (score > 70) return "#16a34a";
  if (score >= 50) return "#d97706";
  return "#dc2626";
}

function Stat({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div style={styles.stat}>
      <span style={{ ...styles.statValue, color }}>{value}</span>
      <span style={styles.statLabel}>{label}</span>
    </div>
  );
}

const styles: Record<string, React.CSSProperties> = {
  container: { padding: "44px 20px 56px", maxWidth: 520, margin: "0 auto" },
  card: { padding: 34, textAlign: "center", display: "grid", justifyItems: "center", gap: 6 },
  emoji: { fontSize: 44 },
  title: { fontSize: 24, fontWeight: 800, marginBottom: 12 },
  scoreCircle: {
    width: 168,
    height: 168,
    borderRadius: 999,
    fontSize: 42,
    fontWeight: 850,
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    margin: "6px auto 22px",
    border: "7px solid",
    fontVariantNumeric: "tabular-nums",
  },
  statsRow: { display: "flex", justifyContent: "center", gap: 32, marginBottom: 12 },
  stat: { display: "flex", flexDirection: "column", alignItems: "center" },
  statValue: { fontSize: 32, fontWeight: 850 },
  statLabel: { color: "var(--text-muted)", fontSize: 13, fontWeight: 600 },
  meta: { color: "var(--text-muted)", marginBottom: 24, fontSize: 14 },
  buttonRow: { display: "flex", justifyContent: "center", gap: 12, flexWrap: "wrap" },
  primaryButton: {
    background: "var(--brand-gradient)",
    color: "#fff",
    border: "none",
    borderRadius: "var(--r-md)",
    padding: "12px 22px",
    fontSize: 16,
    fontWeight: 700,
    cursor: "pointer",
    boxShadow: "var(--shadow-brand)",
  },
  secondaryButton: {
    background: "var(--surface)",
    color: "var(--text)",
    border: "1px solid var(--border-strong)",
    borderRadius: "var(--r-md)",
    padding: "12px 22px",
    fontSize: 16,
    fontWeight: 650,
    cursor: "pointer",
  },
  error: { color: "var(--danger)", marginBottom: 12 },
};
