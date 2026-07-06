"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import {
  getQuiz,
  submitAttempt,
  type AnswerInput,
  type QuizDetail,
} from "../../../../../lib/api";

const OPTION_KEYS = ["a", "b", "c", "d"];

function storageKey(attemptId: string): string {
  return `studyapp.quiz.answers.${attemptId}`;
}

export default function QuizPlayPage() {
  const params = useParams<{ quizId: string; attemptId: string }>();
  const { quizId, attemptId } = params;
  const router = useRouter();

  const [quiz, setQuiz] = useState<QuizDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [current, setCurrent] = useState(0);
  const [answers, setAnswers] = useState<Record<number, number | null>>({});
  const [visited, setVisited] = useState<Record<number, boolean>>({});
  const [timeLeft, setTimeLeft] = useState<number | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [toast, setToast] = useState<string | null>(null);

  const submitLockRef = useRef(false);

  const loadQuiz = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getQuiz(quizId);
      setQuiz(data);
      setTimeLeft(data.timeLimit);
      // Restore any locally-saved answers (offline resilience).
      try {
        const saved = localStorage.getItem(storageKey(attemptId));
        if (saved) {
          setAnswers(JSON.parse(saved) as Record<number, number | null>);
        }
      } catch {
        // ignore malformed local storage
      }
      setVisited({ [data.questions[0]?.id ?? 0]: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load quiz");
    } finally {
      setLoading(false);
    }
  }, [quizId, attemptId]);

  useEffect(() => {
    void loadQuiz();
  }, [loadQuiz]);

  const buildAnswers = useCallback((): AnswerInput[] => {
    if (!quiz) return [];
    return quiz.questions.map((q) => ({
      questionId: q.id,
      selectedOptionId: answers[q.id] ?? null,
    }));
  }, [quiz, answers]);

  const doSubmit = useCallback(async () => {
    if (submitLockRef.current || submitted) return;
    submitLockRef.current = true;
    setSubmitting(true);
    setError(null);
    try {
      const result = await submitAttempt(quizId, attemptId, buildAnswers());
      setSubmitted(true);
      localStorage.removeItem(storageKey(attemptId));
      try {
        localStorage.setItem(`studyapp.quiz.result.${attemptId}`, JSON.stringify(result));
      } catch {
        // ignore storage failure; result page falls back to review data
      }
      router.push(`/quiz/${quizId}/result/${attemptId}`);
    } catch (err) {
      submitLockRef.current = false;
      setSubmitting(false);
      if (typeof navigator !== "undefined" && !navigator.onLine) {
        setToast("No connection — your answers are saved on this device.");
      } else {
        setError(err instanceof Error ? err.message : "Submit failed");
      }
    }
  }, [quizId, attemptId, buildAnswers, router, submitted]);

  // Countdown timer with auto-submit at zero.
  useEffect(() => {
    if (timeLeft === null || submitted) return;
    if (timeLeft <= 0) {
      void doSubmit();
      return;
    }
    const id = setTimeout(() => setTimeLeft((t) => (t === null ? t : t - 1)), 1000);
    return () => clearTimeout(id);
  }, [timeLeft, submitted, doSubmit]);

  const selectOption = useCallback(
    (questionId: number, optionId: number) => {
      if (submitted) return;
      setAnswers((prev) => {
        const next = { ...prev, [questionId]: optionId };
        try {
          localStorage.setItem(storageKey(attemptId), JSON.stringify(next));
        } catch {
          // storage may be unavailable; ignore
        }
        return next;
      });
    },
    [attemptId, submitted]
  );

  const goTo = useCallback(
    (index: number) => {
      if (!quiz) return;
      const clamped = Math.max(0, Math.min(index, quiz.questions.length - 1));
      setCurrent(clamped);
      const qid = quiz.questions[clamped]?.id;
      if (qid != null) setVisited((v) => ({ ...v, [qid]: true }));
    },
    [quiz]
  );

  // Keyboard navigation: A/B/C/D to answer, arrows to move.
  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (!quiz || submitted) return;
      const q = quiz.questions[current];
      if (!q) return;
      const key = e.key.toLowerCase();
      const optIndex = OPTION_KEYS.indexOf(key);
      if (optIndex >= 0 && q.options[optIndex]) {
        selectOption(q.id, q.options[optIndex].id);
      } else if (e.key === "ArrowRight") {
        goTo(current + 1);
      } else if (e.key === "ArrowLeft") {
        goTo(current - 1);
      }
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [quiz, current, submitted, selectOption, goTo]);

  useEffect(() => {
    if (!toast) return;
    const id = setTimeout(() => setToast(null), 3000);
    return () => clearTimeout(id);
  }, [toast]);

  const answeredCount = useMemo(
    () => Object.values(answers).filter((v) => v != null).length,
    [answers]
  );

  function confirmAndSubmit() {
    if (window.confirm(`Submit quiz? You answered ${answeredCount} of ${quiz?.questions.length ?? 0} questions.`)) {
      void doSubmit();
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

  const question = quiz.questions[current];
  const minutes = Math.floor((timeLeft ?? 0) / 60);
  const seconds = (timeLeft ?? 0) % 60;

  return (
    <main style={styles.container}>
      <div style={styles.header}>
        <span style={styles.progressText}>
          Question {current + 1} / {quiz.questions.length}
        </span>
        <span style={{ ...styles.timer, color: (timeLeft ?? 0) <= 30 ? "#b91c1c" : "#111" }}>
          {minutes}:{seconds.toString().padStart(2, "0")}
        </span>
      </div>

      <div style={styles.dots}>
        {quiz.questions.map((q, i) => {
          const isAnswered = answers[q.id] != null;
          const isVisited = visited[q.id];
          const bg = i === current ? "#2563eb" : isAnswered ? "#16a34a" : isVisited ? "#f59e0b" : "#d1d5db";
          return (
            <button
              key={q.id}
              className="btn-reset"
              aria-label={`Go to question ${i + 1}`}
              onClick={() => goTo(i)}
              style={{ ...styles.dot, background: bg }}
            />
          );
        })}
      </div>

      <h2 style={styles.questionText}>{question.text}</h2>

      <div style={styles.options}>
        {question.options.map((opt, idx) => {
          const selected = answers[question.id] === opt.id;
          return (
            <button
              key={opt.id}
              className="btn-reset"
              onClick={() => selectOption(question.id, opt.id)}
              disabled={submitted}
              style={{
                ...styles.option,
                borderColor: selected ? "var(--brand-500)" : "var(--border-strong)",
                background: selected ? "var(--brand-50)" : "var(--surface)",
                color: selected ? "var(--brand-700)" : "var(--text)",
                boxShadow: selected ? "var(--ring)" : "none",
              }}
            >
              <span style={styles.optionLabel}>{opt.label || OPTION_KEYS[idx]?.toUpperCase()}</span>
              <span>{opt.text}</span>
            </button>
          );
        })}
      </div>

      <div style={styles.footer}>
        <button
          className="btn-reset"
          style={styles.footerBtnSecondary}
          onClick={() => goTo(current - 1)}
          disabled={current === 0 || submitted}
        >
          ← Prev
        </button>
        <button
          className="btn-reset"
          style={styles.footerBtnSecondary}
          onClick={() => goTo(current + 1)}
          disabled={current >= quiz.questions.length - 1 || submitted}
        >
          Next →
        </button>
        <button
          style={styles.footerBtnSubmit}
          onClick={confirmAndSubmit}
          disabled={submitting || submitted}
        >
          {submitting ? "Submitting…" : "Submit"}
        </button>
      </div>

      {error ? <p style={styles.error}>{error}</p> : null}
      <p style={styles.hint}>Tip: press A, B, C, or D to answer; arrow keys to navigate.</p>

      {toast ? <div style={styles.toast}>{toast}</div> : null}
    </main>
  );
}

const styles: Record<string, React.CSSProperties> = {
  container: { padding: "32px 20px 56px", maxWidth: 740, margin: "0 auto" },
  header: { display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 18 },
  progressText: { color: "var(--text-muted)", fontWeight: 700, fontSize: 14 },
  timer: {
    fontVariantNumeric: "tabular-nums",
    fontSize: 20,
    fontWeight: 800,
    padding: "4px 12px",
    borderRadius: 999,
    background: "var(--surface)",
    border: "1px solid var(--border)",
    boxShadow: "var(--shadow-xs)",
  },
  footer: { display: "flex", flexDirection: "row", gap: 10, marginTop: 8 },
  footerBtnSecondary: {
    flex: 1,
    background: "var(--surface)",
    color: "var(--text)",
    border: "1px solid var(--border-strong)",
    borderRadius: "var(--r-md)",
    padding: "13px 12px",
    fontSize: 15,
    fontWeight: 650,
    cursor: "pointer",
    minHeight: 48,
  },
  footerBtnSubmit: {
    flex: 1.2,
    background: "var(--brand-gradient)",
    color: "#fff",
    border: "none",
    borderRadius: "var(--r-md)",
    padding: "13px 16px",
    fontSize: 15,
    fontWeight: 800,
    cursor: "pointer",
    boxShadow: "var(--shadow-brand)",
    minHeight: 48,
  },
  dots: { display: "flex", flexWrap: "wrap", gap: 8, marginBottom: 24 },
  dot: { width: 24, height: 24, borderRadius: 999, border: "none", cursor: "pointer", transition: "transform 0.12s ease" },
  questionText: { fontSize: 21, fontWeight: 700, marginBottom: 22, lineHeight: 1.45 },
  options: { display: "grid", gap: 12, marginBottom: 26 },
  option: {
    display: "flex",
    alignItems: "center",
    gap: 12,
    textAlign: "left",
    padding: "15px 16px",
    borderRadius: "var(--r-md)",
    border: "2px solid var(--border-strong)",
    background: "var(--surface)",
    fontSize: 16,
    cursor: "pointer",
    transition: "border-color 0.15s ease, background 0.15s ease, box-shadow 0.15s ease",
  },
  optionLabel: {
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    width: 30,
    height: 30,
    borderRadius: 999,
    background: "var(--brand-50)",
    color: "var(--brand-700)",
    fontWeight: 800,
    flexShrink: 0,
  },
  navRow: { display: "flex", justifyContent: "space-between", gap: 12 },
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
  error: {
    color: "var(--danger)",
    background: "var(--danger-bg)",
    padding: "11px 13px",
    borderRadius: "var(--r-md)",
    marginTop: 12,
    fontWeight: 500,
  },
  hint: { color: "var(--text-subtle)", fontSize: 13, marginTop: 16 },
  toast: {
    position: "fixed",
    bottom: 24,
    left: "50%",
    transform: "translateX(-50%)",
    background: "var(--text)",
    color: "#fff",
    padding: "12px 20px",
    borderRadius: "var(--r-md)",
    fontSize: 14,
    boxShadow: "var(--shadow-lg)",
    fontWeight: 600,
  },
};
