"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import {
  Bar,
  BarChart,
  Cell,
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import {
  getAnalytics,
  getMyReports,
  getTopicAnalytics,
  type Analytics,
  type ReportsPage,
  type TopicAnalytics,
} from "../../lib/api";

const RECENT_LIMIT = 8;

function scoreColor(score: number): string {
  if (score > 70) return "#16a34a";
  if (score >= 50) return "#f59e0b";
  return "#dc2626";
}

function trendBadge(trend: string): { label: string; color: string } {
  switch (trend) {
    case "improving":
      return { label: "▲ Improving", color: "#16a34a" };
    case "declining":
      return { label: "▼ Declining", color: "#dc2626" };
    default:
      return { label: "▬ Stable", color: "#6b7280" };
  }
}

export default function ReportsPageView() {
  const [analytics, setAnalytics] = useState<Analytics | null>(null);
  const [reports, setReports] = useState<ReportsPage | null>(null);
  const [topics, setTopics] = useState<TopicAnalytics | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadAll = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [a, r, t] = await Promise.all([
        getAnalytics(),
        getMyReports(1, RECENT_LIMIT),
        getTopicAnalytics(),
      ]);
      setAnalytics(a);
      setReports(r);
      setTopics(t);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load analytics");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  if (loading) {
    return (
      <main style={styles.container}>
        <h1 style={styles.title}>Performance Analytics</h1>
        <p style={styles.muted}>Loading your analytics…</p>
      </main>
    );
  }

  if (error) {
    return (
      <main style={styles.container}>
        <h1 style={styles.title}>Performance Analytics</h1>
        <p style={styles.error}>{error}</p>
        <button className="btn-reset" style={styles.secondaryButton} onClick={() => void loadAll()}>
          Retry
        </button>
      </main>
    );
  }

  const summary = analytics?.summary;
  const hasData = !!summary && summary.totalQuizzes > 0;

  if (!hasData) {
    return (
      <main style={styles.container}>
        <h1 style={styles.title}>Performance Analytics</h1>
        <EmptyState
          title="No quiz data yet"
          message="Complete a quiz to unlock your performance analytics — trends, subject strengths, and weak topics will appear here."
        />
      </main>
    );
  }

  const improvement = summary.improvement;
  const weakest = (topics?.topics ?? []).slice(0, 5);

  return (
    <main style={styles.container} className="animate-in">
      <h1 style={styles.title}>Performance Analytics</h1>
      <p style={styles.lead}>Your progress at a glance — trends, strengths, and topics to work on.</p>

      {/* Row 1 — Summary cards */}
      <section style={styles.cardGrid}>
        <StatCard label="Total Quizzes" value={String(summary.totalQuizzes)} />
        <StatCard label="Average Score" value={`${summary.averageScore.toFixed(1)}%`} color={scoreColor(summary.averageScore)} />
        <StatCard label="Study Streak" value={`${summary.studyStreakDays} day${summary.studyStreakDays === 1 ? "" : "s"}`} />
        <StatCard
          label="This vs Last Week"
          value={`${summary.thisWeekScore.toFixed(0)}% vs ${summary.lastWeekScore.toFixed(0)}%`}
          sub={`${improvement >= 0 ? "+" : ""}${improvement.toFixed(1)} pts`}
          color={improvement >= 0 ? "#16a34a" : "#dc2626"}
        />
      </section>

      {/* Row 2 — Weekly score trend */}
      <section style={styles.panel}>
        <h2 style={styles.panelTitle}>Score Trend (last 8 weeks)</h2>
        {analytics && analytics.weeklyScores.length > 0 ? (
          <ResponsiveContainer width="100%" height={260}>
            <LineChart data={analytics.weeklyScores} margin={{ top: 8, right: 16, bottom: 8, left: -16 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#eee" />
              <XAxis dataKey="week" tick={{ fontSize: 12 }} />
              <YAxis domain={[0, 100]} tick={{ fontSize: 12 }} />
              <Tooltip formatter={(v: number) => [`${v}%`, "Score"]} />
              <Line type="monotone" dataKey="score" stroke="#2563eb" strokeWidth={2} dot={{ r: 3 }} />
            </LineChart>
          </ResponsiveContainer>
        ) : (
          <EmptyState title="Not enough history" message="Complete quizzes across multiple weeks to see your trend." small />
        )}
      </section>

      {/* Row 3 — Subject breakdown */}
      <section style={styles.panel}>
        <h2 style={styles.panelTitle}>Subject Performance</h2>
        {analytics && analytics.subjectBreakdown.length > 0 ? (
          <ResponsiveContainer width="100%" height={260}>
            <BarChart data={analytics.subjectBreakdown} margin={{ top: 8, right: 16, bottom: 8, left: -16 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="#eee" />
              <XAxis dataKey="subject" tick={{ fontSize: 12 }} />
              <YAxis domain={[0, 100]} tick={{ fontSize: 12 }} />
              <Tooltip formatter={(v: number) => [`${v}%`, "Avg Score"]} />
              <Bar dataKey="avgScore" radius={[4, 4, 0, 0]}>
                {analytics.subjectBreakdown.map((s) => (
                  <Cell key={s.subject} fill={scoreColor(s.avgScore)} />
                ))}
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        ) : (
          <EmptyState title="No subjects yet" message="Subject performance appears once you complete quizzes." small />
        )}
      </section>

      {/* Row 4 — Recent quizzes table */}
      <section style={styles.panel}>
        <h2 style={styles.panelTitle}>Recent Quizzes</h2>
        {reports && reports.reports.length > 0 ? (
          <table style={styles.table}>
            <thead>
              <tr>
                <th style={styles.th}>Date</th>
                <th style={styles.th}>Quiz</th>
                <th style={styles.th}>Score</th>
                <th style={styles.th}></th>
              </tr>
            </thead>
            <tbody>
              {reports.reports.map((r) => (
                <tr key={r.attemptId} style={styles.tr}>
                  <td style={styles.td}>{r.completedAt ? new Date(r.completedAt).toLocaleDateString() : "—"}</td>
                  <td style={styles.td}>{r.quizTitle}</td>
                  <td style={styles.td}>
                    <span style={{ ...styles.scoreBadge, background: scoreColor(r.score) }}>{r.score.toFixed(0)}%</span>
                  </td>
                  <td style={styles.td}>
                    <Link href={`/quiz/${r.quizId}/review/${r.attemptId}`} style={styles.reviewLink}>
                      Review
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <EmptyState title="No recent quizzes" message="Your latest attempts will show up here." small />
        )}
      </section>

      {/* Row 5 — Weak topics */}
      <section style={styles.panel}>
        <h2 style={styles.panelTitle}>Weakest Topics</h2>
        {weakest.length > 0 ? (
          <ul style={styles.topicList}>
            {weakest.map((t) => (
              <li key={`${t.subject}-${t.topic}`} style={styles.topicItem}>
                <div>
                  <p style={styles.topicName}>{t.topic}</p>
                  <p style={styles.topicSub}>
                    {t.subject ? `${t.subject} · ` : ""}
                    {t.correctCount}/{t.totalAnswered} correct
                  </p>
                </div>
                <div style={styles.itemRight}>
                  <span style={{ ...styles.scoreBadge, background: scoreColor(t.accuracy) }}>
                    {t.accuracy.toFixed(0)}%
                  </span>
                  {t.sampleQuizId ? (
                    <Link href={`/quiz/${t.sampleQuizId}`} style={styles.practiceButton}>
                      Practice this
                    </Link>
                  ) : null}
                </div>
              </li>
            ))}
          </ul>
        ) : (
          <EmptyState title="No topic data yet" message="Answer more questions to reveal your weak topics." small />
        )}
      </section>
    </main>
  );
}

function StatCard({ label, value, sub, color }: { label: string; value: string; sub?: string; color?: string }) {
  return (
    <div style={styles.statCard}>
      <p style={styles.statLabel}>{label}</p>
      <p style={{ ...styles.statValue, color: color ?? "#111" }}>{value}</p>
      {sub ? <p style={{ ...styles.statSub, color: color ?? "#6b7280" }}>{sub}</p> : null}
    </div>
  );
}

function EmptyState({ title, message, small }: { title: string; message: string; small?: boolean }) {
  return (
    <div style={{ ...styles.emptyState, padding: small ? 24 : 48 }}>
      <div style={styles.emptyIcon} aria-hidden>
        📊
      </div>
      <p style={styles.emptyTitle}>{title}</p>
      <p style={styles.emptyMessage}>{message}</p>
    </div>
  );
}

const styles: Record<string, React.CSSProperties> = {
  container: { padding: "36px 20px 56px", maxWidth: 980, margin: "0 auto" },
  title: { fontSize: 30, fontWeight: 850, marginBottom: 4 },
  lead: { color: "var(--text-muted)", fontSize: 15, marginTop: 0, marginBottom: 26 },
  muted: { color: "var(--text-muted)" },
  error: { color: "var(--danger)", marginBottom: 12 },
  cardGrid: {
    display: "grid",
    gridTemplateColumns: "repeat(auto-fit, minmax(180px, 1fr))",
    gap: 16,
    marginBottom: 24,
  },
  statCard: {
    border: "1px solid var(--border)",
    borderRadius: "var(--r-lg)",
    padding: 20,
    background: "var(--surface)",
    boxShadow: "var(--shadow-sm)",
  },
  statLabel: { color: "var(--text-muted)", fontSize: 13, marginBottom: 8, fontWeight: 600 },
  statValue: { fontSize: 26, fontWeight: 800 },
  statSub: { fontSize: 13, fontWeight: 700, marginTop: 4 },
  panel: {
    border: "1px solid var(--border)",
    borderRadius: "var(--r-lg)",
    padding: 22,
    marginBottom: 22,
    background: "var(--surface)",
    boxShadow: "var(--shadow-sm)",
  },
  panelTitle: { fontSize: 17, fontWeight: 750, marginBottom: 16 },
  table: { width: "100%", borderCollapse: "collapse" },
  th: { textAlign: "left", fontSize: 12, color: "var(--text-muted)", fontWeight: 700, padding: "8px 10px", borderBottom: "1px solid var(--border)", textTransform: "uppercase", letterSpacing: "0.03em" },
  tr: { borderBottom: "1px solid var(--surface-2)" },
  td: { padding: "12px 10px", fontSize: 14 },
  scoreBadge: { color: "#fff", fontWeight: 700, padding: "4px 10px", borderRadius: 999, fontSize: 13 },
  reviewLink: { color: "var(--brand-600)", fontWeight: 700, textDecoration: "none" },
  topicList: { listStyle: "none", padding: 0, display: "grid", gap: 10, margin: 0 },
  topicItem: {
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    border: "1px solid var(--border)",
    borderRadius: "var(--r-md)",
    padding: 14,
    background: "var(--surface-2)",
  },
  topicName: { fontWeight: 700 },
  topicSub: { color: "var(--text-muted)", fontSize: 13, marginTop: 2 },
  itemRight: { display: "flex", alignItems: "center", gap: 12 },
  practiceButton: {
    background: "var(--brand-gradient)",
    color: "#fff",
    fontWeight: 700,
    fontSize: 13,
    padding: "7px 13px",
    borderRadius: "var(--r-sm)",
    textDecoration: "none",
    boxShadow: "var(--shadow-brand)",
  },
  emptyState: { textAlign: "center", color: "var(--text-muted)", border: "1px dashed var(--border-strong)", borderRadius: "var(--r-lg)", background: "var(--surface-2)" },
  emptyIcon: { fontSize: 40, marginBottom: 8 },
  emptyTitle: { fontWeight: 700, color: "var(--text)", marginBottom: 4 },
  emptyMessage: { fontSize: 14 },
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
};
