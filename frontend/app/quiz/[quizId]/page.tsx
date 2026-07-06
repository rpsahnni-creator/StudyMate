"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { getQuiz, startAttempt, type QuizDetail } from "../../../lib/api";

export default function QuizStartPage() {
  const params = useParams<{ quizId: string }>();
  const quizId = params.quizId;
  const router = useRouter();

  const [quiz, setQuiz] = useState<QuizDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [starting, setStarting] = useState(false);

  const loadQuiz = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getQuiz(quizId);
      setQuiz(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load quiz");
    } finally {
      setLoading(false);
    }
  }, [quizId]);

  useEffect(() => {
    void loadQuiz();
  }, [loadQuiz]);

  async function handleStart() {
    setStarting(true);
    setError(null);
    try {
      const { attemptId } = await startAttempt(quizId);
      router.push(`/quiz/${quizId}/play/${attemptId}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not start quiz");
      setStarting(false);
    }
  }

  if (loading) {
    return (
      <main style={styles.container}>
        <p>Loading quiz…</p>
      </main>
    );
  }

  if (error && !quiz) {
    return (
      <main style={styles.container}>
        <p style={styles.error}>{error}</p>
        <button style={styles.primaryButton} onClick={() => void loadQuiz()}>
          Retry
        </button>
      </main>
    );
  }

  if (!quiz) return null;

  return (
    <main style={styles.container} className="animate-in">
      <div style={styles.card} className="card">
        <span style={styles.eyebrow}>Ready to begin?</span>
        <h1 style={styles.title}>{quiz.title}</h1>
        <div style={styles.metaRow}>
          <span style={styles.badge}>{quiz.subject || "General"}</span>
          <span style={styles.badge}>{quiz.board || "—"}</span>
        </div>

        <div style={styles.infoGrid}>
          <div style={styles.infoBox}>
            <span style={styles.infoValue}>{quiz.totalQuestions}</span>
            <span style={styles.infoLabel}>Questions</span>
          </div>
          <div style={styles.infoBox}>
            <span style={styles.infoValue}>{Math.round(quiz.timeLimit / 60)}</span>
            <span style={styles.infoLabel}>Minutes</span>
          </div>
        </div>

        {error ? <p style={styles.error}>{error}</p> : null}

        <button style={styles.primaryButton} onClick={() => void handleStart()} disabled={starting}>
          {starting ? "Starting…" : "Start Quiz →"}
        </button>
      </div>
    </main>
  );
}

const styles: Record<string, React.CSSProperties> = {
  container: { padding: "40px 20px 56px", maxWidth: 620, margin: "0 auto" },
  card: { padding: 30, display: "grid", gap: 14 },
  eyebrow: {
    justifySelf: "start",
    padding: "5px 12px",
    borderRadius: 999,
    background: "var(--brand-50)",
    color: "var(--brand-700)",
    fontSize: 13,
    fontWeight: 700,
    border: "1px solid var(--brand-100)",
  },
  title: { fontSize: 28, fontWeight: 800, margin: 0 },
  metaRow: { display: "flex", gap: 8, flexWrap: "wrap" },
  badge: {
    background: "var(--brand-50)",
    color: "var(--brand-700)",
    padding: "5px 12px",
    borderRadius: 999,
    fontSize: 13,
    fontWeight: 700,
  },
  infoGrid: { display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12, marginTop: 4 },
  infoBox: {
    display: "grid",
    gap: 2,
    padding: 16,
    borderRadius: "var(--r-md)",
    background: "var(--surface-2)",
    border: "1px solid var(--border)",
    textAlign: "center",
  },
  infoValue: { fontSize: 28, fontWeight: 850, color: "var(--text)" },
  infoLabel: { fontSize: 13, color: "var(--text-muted)", fontWeight: 600 },
  primaryButton: {
    marginTop: 6,
    padding: "13px 20px",
    fontSize: 16,
  },
  error: {
    color: "var(--danger)",
    background: "var(--danger-bg)",
    padding: "11px 13px",
    borderRadius: "var(--r-md)",
    margin: 0,
    fontWeight: 500,
  },
};
