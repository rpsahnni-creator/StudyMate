"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { getReview, type ReviewDetail, type ReviewQuestion } from "../../../../../lib/api";

export default function QuizReviewPage() {
  const params = useParams<{ quizId: string; attemptId: string }>();
  const { quizId, attemptId } = params;
  const router = useRouter();

  const [review, setReview] = useState<ReviewDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<Record<number, boolean>>({});

  const loadReview = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getReview(quizId, attemptId);
      setReview(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load review");
    } finally {
      setLoading(false);
    }
  }, [quizId, attemptId]);

  useEffect(() => {
    void loadReview();
  }, [loadReview]);

  if (loading) {
    return (
      <main style={styles.container}>
        <p>Loading review…</p>
      </main>
    );
  }

  if (error || !review) {
    return (
      <main style={styles.container}>
        <p style={styles.error}>{error ?? "No review found"}</p>
        <button style={styles.primaryButton} onClick={() => void loadReview()}>
          Retry
        </button>
      </main>
    );
  }

  const correctCount = review.questions.filter((q) => q.status === "correct").length;

  return (
    <main style={styles.container}>
      <div style={styles.summary} className="card">
        <h1 style={styles.title}>Review</h1>
        <p style={styles.summaryText}>
          Score <strong>{review.score.toFixed(1)}%</strong> · {correctCount} / {review.questions.length} correct
        </p>
        <button className="btn-reset" style={styles.secondaryButton} onClick={() => router.push(`/quiz/${quizId}/result/${attemptId}`)}>
          ← Back to result
        </button>
      </div>

      <ol style={styles.list}>
        {review.questions.map((q, index) => (
          <ReviewCard
            key={q.id}
            question={q}
            index={index}
            expanded={!!expanded[q.id]}
            onToggle={() => setExpanded((e) => ({ ...e, [q.id]: !e[q.id] }))}
          />
        ))}
      </ol>
    </main>
  );
}

function ReviewCard({
  question,
  index,
  expanded,
  onToggle,
}: {
  question: ReviewQuestion;
  index: number;
  expanded: boolean;
  onToggle: () => void;
}) {
  const statusColor =
    question.status === "correct" ? "#16a34a" : question.status === "wrong" ? "#dc2626" : "#d97706";

  return (
    <li style={{ ...styles.card, borderLeft: `4px solid ${statusColor}` }}>
      <div style={styles.cardHeader}>
        <span style={styles.qNumber}>Q{index + 1}</span>
        <span style={{ ...styles.statusPill, background: statusColor }}>{question.status}</span>
      </div>
      <p style={styles.qText}>{question.text}</p>

      <div style={styles.answerRow}>
        <span style={styles.answerLabel}>Your answer:</span>
        <span
          style={{
            ...styles.answerValue,
            color: question.status === "correct" ? "#16a34a" : question.status === "wrong" ? "#dc2626" : "#666",
          }}
        >
          {question.yourAnswer != null ? `Option #${question.yourAnswer}` : "Skipped"}
        </span>
      </div>

      {!question.isCorrect && question.correctAnswer != null ? (
        <div style={styles.answerRow}>
          <span style={styles.answerLabel}>Correct answer:</span>
          <span style={{ ...styles.answerValue, color: "#16a34a" }}>Option #{question.correctAnswer}</span>
        </div>
      ) : null}

      {question.explanation ? (
        <div style={styles.explanationWrap}>
          <button className="btn-reset" style={styles.explanationToggle} onClick={onToggle}>
            {expanded ? "Hide explanation" : "Show explanation"}
          </button>
          {expanded ? <p style={styles.explanationText}>{question.explanation}</p> : null}
        </div>
      ) : null}
    </li>
  );
}

const styles: Record<string, React.CSSProperties> = {
  container: { padding: "32px 20px 56px", maxWidth: 740, margin: "0 auto" },
  summary: { marginBottom: 24, padding: 22 },
  title: { fontSize: 26, fontWeight: 800, marginBottom: 8 },
  summaryText: { color: "var(--text-muted)", marginBottom: 14, fontSize: 15 },
  list: { listStyle: "none", padding: 0, display: "grid", gap: 14 },
  card: {
    border: "1px solid var(--border)",
    borderRadius: "var(--r-lg)",
    padding: 18,
    background: "var(--surface)",
    boxShadow: "var(--shadow-sm)",
  },
  cardHeader: { display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 10 },
  qNumber: { fontWeight: 800, color: "var(--text-muted)" },
  statusPill: {
    color: "#fff",
    padding: "3px 11px",
    borderRadius: 999,
    fontSize: 12,
    fontWeight: 700,
    textTransform: "capitalize",
  },
  qText: { fontSize: 16.5, fontWeight: 700, marginBottom: 14, lineHeight: 1.45 },
  answerRow: { display: "flex", gap: 8, marginBottom: 5 },
  answerLabel: { color: "var(--text-muted)" },
  answerValue: { fontWeight: 700 },
  explanationWrap: { marginTop: 14 },
  explanationToggle: {
    background: "none",
    border: "none",
    color: "var(--brand-600)",
    cursor: "pointer",
    padding: 0,
    fontSize: 14,
    fontWeight: 700,
  },
  explanationText: {
    marginTop: 10,
    color: "var(--text)",
    lineHeight: 1.6,
    background: "var(--brand-gradient-soft)",
    border: "1px solid var(--brand-100)",
    padding: 14,
    borderRadius: "var(--r-md)",
  },
  primaryButton: {
    background: "var(--brand-gradient)",
    color: "#fff",
    border: "none",
    borderRadius: "var(--r-md)",
    padding: "12px 20px",
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
    padding: "10px 16px",
    fontSize: 14,
    fontWeight: 650,
    cursor: "pointer",
  },
  error: { color: "var(--danger)", marginBottom: 12 },
};
