"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "../../components/AuthProvider";
import {
  abandonGoal,
  getGoals,
  getMyGoal,
  selectGoal,
  type CareerGoal,
  type MyGoal,
} from "../../lib/goals";

export default function GoalsPage() {
  const router = useRouter();
  const { isLoggedIn, isLoading: authLoading } = useAuth();

  const [goals, setGoals] = useState<CareerGoal[]>([]);
  const [myGoal, setMyGoal] = useState<MyGoal | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actingGoalId, setActingGoalId] = useState<number | null>(null);
  const [changingGoal, setChangingGoal] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const catalog = await getGoals();
      setGoals(catalog);
      if (isLoggedIn) {
        const active = await getMyGoal();
        setMyGoal(active);
      } else {
        setMyGoal(null);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load career goals");
    } finally {
      setLoading(false);
    }
  }, [isLoggedIn]);

  useEffect(() => {
    if (authLoading) return;
    void load();
  }, [authLoading, load]);

  async function handleSelect(goalId: number) {
    if (!isLoggedIn) {
      router.push(`/auth/login?from=${encodeURIComponent("/goals")}`);
      return;
    }
    setActingGoalId(goalId);
    setError(null);
    try {
      await selectGoal(goalId);
      await load();
      setChangingGoal(false);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not select this goal");
    } finally {
      setActingGoalId(null);
    }
  }

  async function handleAbandon() {
    if (!window.confirm("Stop tracking this goal? Your practice history is kept.")) return;
    setError(null);
    try {
      await abandonGoal();
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not update your goal");
    }
  }

  if (authLoading || loading) {
    return (
      <main style={styles.page}>
        <p style={styles.muted}>Loading career goals…</p>
      </main>
    );
  }

  const showCatalog = !myGoal || changingGoal;

  return (
    <main style={styles.page} className="animate-in">
      <header style={styles.header}>
        <span style={styles.eyebrow}>🎯 Career Goals</span>
        <h1 style={styles.title}>Set a target and get a daily practice path</h1>
        <p style={styles.subtitle}>
          Pick an exam or goal — we&apos;ll track your streak and surface your weakest
          topics every day until you get there.
        </p>
      </header>

      {error ? <p style={styles.error}>{error}</p> : null}

      {myGoal && !showCatalog ? (
        <ActiveGoalCard goal={myGoal} onChangeGoal={() => setChangingGoal(true)} onAbandon={() => void handleAbandon()} />
      ) : (
        <section style={styles.grid}>
          {goals.length === 0 ? (
            <p style={styles.muted}>No career goals are available right now — check back soon.</p>
          ) : (
            goals.map((goal) => (
              <GoalCard
                key={goal.id}
                goal={goal}
                isCurrent={myGoal?.goalId === goal.id}
                busy={actingGoalId === goal.id}
                onSelect={() => void handleSelect(goal.id)}
              />
            ))
          )}
          {changingGoal ? (
            <button type="button" className="btn-reset" style={styles.cancelChange} onClick={() => setChangingGoal(false)}>
              ← Keep my current goal
            </button>
          ) : null}
        </section>
      )}

      {!isLoggedIn ? (
        <p style={styles.loginHint}>
          <Link href={`/auth/login?from=${encodeURIComponent("/goals")}`} style={styles.link}>
            Log in
          </Link>{" "}
          to select a goal and start tracking daily practice.
        </p>
      ) : null}
    </main>
  );
}

function GoalCard({
  goal,
  isCurrent,
  busy,
  onSelect,
}: {
  goal: CareerGoal;
  isCurrent: boolean;
  busy: boolean;
  onSelect: () => void;
}) {
  return (
    <article style={{ ...styles.card, ...(isCurrent ? styles.cardCurrent : {}) }}>
      <h2 style={styles.cardTitle}>{goal.name}</h2>
      {goal.examName ? <p style={styles.cardExam}>{goal.examName}</p> : null}
      {goal.description ? <p style={styles.cardDesc}>{goal.description}</p> : null}
      <div style={styles.cardMeta}>
        <span style={styles.metaChip}>{goal.targetMonths} month prep</span>
        {goal.subjectAreas.slice(0, 3).map((s) => (
          <span key={s} style={styles.metaChip}>
            {s}
          </span>
        ))}
      </div>
      <button
        type="button"
        className="btn-reset"
        style={isCurrent ? styles.disabledBtn : styles.selectBtn}
        disabled={isCurrent || busy}
        onClick={onSelect}
      >
        {isCurrent ? "Current goal" : busy ? "Selecting…" : "Select this goal"}
      </button>
    </article>
  );
}

function ActiveGoalCard({
  goal,
  onChangeGoal,
  onAbandon,
}: {
  goal: MyGoal;
  onChangeGoal: () => void;
  onAbandon: () => void;
}) {
  return (
    <section style={styles.activeCard} className="card">
      <div style={styles.activeHeader}>
        <div>
          <span style={styles.eyebrowSmall}>Your active goal</span>
          <h2 style={styles.activeTitle}>{goal.name}</h2>
          {goal.examName ? <p style={styles.cardExam}>{goal.examName}</p> : null}
        </div>
        {goal.daysRemaining != null ? (
          <div style={styles.daysBadge}>
            <span style={styles.daysValue}>{goal.daysRemaining}</span>
            <span style={styles.daysLabel}>days left</span>
          </div>
        ) : null}
      </div>

      <dl style={styles.statsGrid}>
        <StatBlock label="Current streak" value={`${goal.progress.currentStreak} day${goal.progress.currentStreak === 1 ? "" : "s"}`} />
        <StatBlock label="Completed sets" value={`${goal.progress.completedPractices}/${goal.progress.totalPractices}`} />
        <StatBlock
          label="Average score"
          value={goal.progress.averageScore != null ? `${goal.progress.averageScore.toFixed(0)}%` : "—"}
        />
        <StatBlock label="Today" value={todayStatusLabel(goal.progress.todayStatus)} />
      </dl>

      <div style={styles.activeActions}>
        <button type="button" className="btn-reset" style={styles.secondaryBtn} onClick={onChangeGoal}>
          Change goal
        </button>
        <button type="button" className="btn-reset" style={styles.dangerBtn} onClick={onAbandon}>
          Stop tracking
        </button>
      </div>
    </section>
  );
}

function StatBlock({ label, value }: { label: string; value: string }) {
  return (
    <div style={styles.statBlock}>
      <dt style={styles.statLabel}>{label}</dt>
      <dd style={styles.statValue}>{value}</dd>
    </div>
  );
}

function todayStatusLabel(status: MyGoal["progress"]["todayStatus"]): string {
  switch (status) {
    case "completed":
      return "Done ✓";
    case "pending":
      return "In progress";
    default:
      return "Not started";
  }
}

const styles: Record<string, React.CSSProperties> = {
  page: { padding: "36px 20px 56px", maxWidth: 900, margin: "0 auto", display: "grid", gap: 24 },
  header: { display: "grid", gap: 10 },
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
  title: { margin: 0, fontSize: 28, fontWeight: 800 },
  subtitle: { margin: 0, color: "var(--text-muted)", fontSize: 15, lineHeight: 1.6, maxWidth: 640 },
  muted: { color: "var(--text-muted)" },
  error: {
    color: "var(--danger)",
    background: "var(--danger-bg)",
    padding: "11px 13px",
    borderRadius: "var(--r-md)",
    margin: 0,
    fontWeight: 500,
  },
  grid: {
    display: "grid",
    gridTemplateColumns: "repeat(auto-fit, minmax(240px, 1fr))",
    gap: 16,
    alignItems: "start",
  },
  card: {
    border: "1px solid var(--border)",
    borderRadius: "var(--r-lg)",
    padding: 20,
    background: "var(--surface)",
    boxShadow: "var(--shadow-sm)",
    display: "grid",
    gap: 8,
  },
  cardCurrent: { outline: "2px solid var(--brand-500)", outlineOffset: 2 },
  cardTitle: { margin: 0, fontSize: 18, fontWeight: 750 },
  cardExam: { margin: 0, color: "var(--brand-700)", fontSize: 13, fontWeight: 700 },
  cardDesc: { margin: 0, color: "var(--text-muted)", fontSize: 13.5, lineHeight: 1.5 },
  cardMeta: { display: "flex", flexWrap: "wrap", gap: 6, marginTop: 2 },
  metaChip: {
    fontSize: 12,
    fontWeight: 600,
    padding: "3px 9px",
    borderRadius: 999,
    background: "var(--surface-2)",
    color: "var(--text-muted)",
    border: "1px solid var(--border)",
  },
  selectBtn: {
    marginTop: 8,
    border: "none",
    borderRadius: "var(--r-md)",
    padding: "10px 16px",
    background: "var(--brand-gradient)",
    color: "#fff",
    fontWeight: 700,
    cursor: "pointer",
    fontSize: 14,
  },
  disabledBtn: {
    marginTop: 8,
    border: "1px solid var(--border)",
    borderRadius: "var(--r-md)",
    padding: "10px 16px",
    background: "var(--surface-2)",
    color: "var(--text-subtle)",
    fontWeight: 650,
    cursor: "not-allowed",
    fontSize: 14,
  },
  cancelChange: {
    alignSelf: "start",
    color: "var(--brand-600)",
    fontWeight: 700,
    fontSize: 14,
    padding: "10px 4px",
    cursor: "pointer",
  },
  loginHint: { color: "var(--text-muted)", fontSize: 14 },
  link: { color: "var(--brand-600)", fontWeight: 700 },
  activeCard: { padding: 24, display: "grid", gap: 18 },
  activeHeader: { display: "flex", justifyContent: "space-between", alignItems: "flex-start", gap: 16, flexWrap: "wrap" },
  eyebrowSmall: { fontSize: 12, fontWeight: 700, color: "var(--brand-700)", textTransform: "uppercase", letterSpacing: "0.04em" },
  activeTitle: { margin: "4px 0 2px", fontSize: 24, fontWeight: 800 },
  daysBadge: {
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
    padding: "10px 16px",
    borderRadius: "var(--r-md)",
    background: "var(--brand-gradient-soft)",
    border: "1px solid var(--brand-100)",
  },
  daysValue: { fontSize: 22, fontWeight: 850, color: "var(--brand-700)" },
  daysLabel: { fontSize: 11, color: "var(--text-muted)", fontWeight: 600 },
  statsGrid: {
    display: "grid",
    gridTemplateColumns: "repeat(auto-fit, minmax(140px, 1fr))",
    gap: 12,
    margin: 0,
  },
  statBlock: {
    padding: 14,
    borderRadius: "var(--r-md)",
    background: "var(--surface-2)",
    border: "1px solid var(--border)",
  },
  statLabel: { margin: 0, fontSize: 12, color: "var(--text-muted)", fontWeight: 600 },
  statValue: { margin: "4px 0 0", fontSize: 20, fontWeight: 800 },
  activeActions: { display: "flex", gap: 10, flexWrap: "wrap" },
  secondaryBtn: {
    border: "1px solid var(--border-strong)",
    background: "var(--surface)",
    borderRadius: "var(--r-md)",
    padding: "10px 16px",
    cursor: "pointer",
    fontSize: 14,
    fontWeight: 650,
    color: "var(--text)",
  },
  dangerBtn: {
    border: "1px solid var(--danger)",
    background: "var(--danger-bg)",
    borderRadius: "var(--r-md)",
    padding: "10px 16px",
    cursor: "pointer",
    fontSize: 14,
    fontWeight: 650,
    color: "var(--danger)",
  },
};
