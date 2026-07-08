import React, { useCallback, useMemo, useState } from "react";
import {
  ActivityIndicator,
  Dimensions,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import { router, useFocusEffect } from "expo-router";
import { Ionicons } from "@expo/vector-icons";
import { LineChart } from "react-native-chart-kit";
import Svg, { Circle, G } from "react-native-svg";
import { SkyBackground } from "../../components/SkyBackground";
import {
  getAnalytics,
  getMyReports,
  getTopicAnalytics,
  type Analytics,
  type DailyActivity,
  type ReportItem,
  type SubjectBreakdown,
  type TopicAccuracy,
} from "../../lib/api";
import { skyScreen } from "../../lib/skyScreen";
import { colors, radius, shadow, spacing, scoreColor } from "../../lib/theme";

const RECENT_LIMIT = 8;
const screenWidth = Dimensions.get("window").width;
const CONTENT_PAD = spacing.xl; // matches skyScreen.content padding
const chartWidth = screenWidth - CONTENT_PAD * 2 - 16;

type IoniconName = keyof typeof Ionicons.glyphMap;

// --- small pure helpers ---

function clampPct(n: number): number {
  if (!Number.isFinite(n)) return 0;
  return Math.max(0, Math.min(100, n));
}

function accuracyOf(correct: number, attempted: number): number {
  if (attempted <= 0) return 0;
  return clampPct((correct / attempted) * 100);
}

function performanceBand(score: number): { label: string; color: string; icon: IoniconName } {
  if (score >= 85) return { label: "Outstanding", color: colors.success, icon: "trophy" };
  if (score >= 70) return { label: "On track", color: colors.brand, icon: "trending-up" };
  if (score >= 50) return { label: "Building up", color: colors.warning, icon: "barbell" };
  return { label: "Needs focus", color: colors.danger, icon: "flame" };
}

function trendMeta(trend: string): { label: string; color: string; icon: IoniconName } {
  switch (trend) {
    case "improving":
      return { label: "Improving", color: colors.success, icon: "arrow-up" };
    case "declining":
      return { label: "Declining", color: colors.danger, icon: "arrow-down" };
    default:
      return { label: "Stable", color: colors.textMuted, icon: "remove" };
  }
}

// Build a fixed 14-day window (oldest → newest) keyed by UTC date to match backend.
function last14Days(activity: DailyActivity[]): { key: string; day: number; data: DailyActivity | null }[] {
  const map = new Map(activity.map((a) => [a.date, a]));
  const out: { key: string; day: number; data: DailyActivity | null }[] = [];
  const now = new Date();
  for (let i = 13; i >= 0; i--) {
    const d = new Date(now);
    d.setDate(now.getDate() - i);
    const key = d.toISOString().slice(0, 10);
    out.push({ key, day: d.getDate(), data: map.get(key) ?? null });
  }
  return out;
}

// Auto-generated coaching insights from the analytics payload.
function buildInsights(
  analytics: Analytics,
  topics: TopicAccuracy[]
): { icon: IoniconName; color: string; text: string }[] {
  const out: { icon: IoniconName; color: string; text: string }[] = [];
  const s = analytics.summary;

  if (s.improvement > 0.5) {
    out.push({
      icon: "rocket",
      color: colors.success,
      text: `You're up ${s.improvement.toFixed(1)} pts vs last week — momentum is on your side.`,
    });
  } else if (s.improvement < -0.5) {
    out.push({
      icon: "alert-circle",
      color: colors.warning,
      text: `Down ${Math.abs(s.improvement).toFixed(1)} pts vs last week — a short review could turn it around.`,
    });
  }

  const subjects = [...analytics.subjectBreakdown].sort((a, b) => b.avgScore - a.avgScore);
  if (subjects.length > 0) {
    const best = subjects[0];
    out.push({
      icon: "ribbon",
      color: colors.brand,
      text: `Strongest subject: ${best.subject} at ${best.avgScore.toFixed(0)}%.`,
    });
  }

  const focus = topics.find((t) => t.totalAnswered >= 2) ?? topics[0];
  if (focus) {
    out.push({
      icon: "locate",
      color: colors.danger,
      text: `Focus next on “${focus.topic}” (${focus.accuracy.toFixed(0)}% accuracy).`,
    });
  }

  if (s.studyStreakDays >= 3) {
    out.push({
      icon: "flame",
      color: "#f97316",
      text: `${s.studyStreakDays}-day streak going strong — keep it alive today!`,
    });
  }

  return out.slice(0, 3);
}

function ScreenWrap({ children }: { children: React.ReactNode }) {
  return <SkyBackground>{children}</SkyBackground>;
}

export default function ReportsScreen() {
  const [analytics, setAnalytics] = useState<Analytics | null>(null);
  const [reports, setReports] = useState<ReportItem[]>([]);
  const [topics, setTopics] = useState<TopicAccuracy[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadAll = useCallback(async (isRefresh = false) => {
    if (isRefresh) setRefreshing(true);
    else setLoading(true);
    setError(null);
    try {
      const [a, r, t] = await Promise.all([
        getAnalytics(),
        getMyReports(1, RECENT_LIMIT),
        getTopicAnalytics(),
      ]);
      setAnalytics(a);
      setReports(r.reports);
      setTopics(t.topics);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load analytics");
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, []);

  useFocusEffect(
    useCallback(() => {
      void loadAll();
    }, [loadAll])
  );

  const insights = useMemo(
    () => (analytics ? buildInsights(analytics, topics) : []),
    [analytics, topics]
  );

  if (loading) {
    return (
      <ScreenWrap>
        <View style={skyScreen.center}>
          <ActivityIndicator size="large" color={colors.brand} />
          <Text style={styles.muted}>Loading your performance…</Text>
        </View>
      </ScreenWrap>
    );
  }

  if (error) {
    return (
      <ScreenWrap>
        <View style={skyScreen.center}>
          <Ionicons name="cloud-offline-outline" size={40} color={colors.textSubtle} />
          <Text style={styles.error}>{error}</Text>
          <TouchableOpacity style={styles.retryButton} onPress={() => void loadAll()}>
            <Text style={styles.retryText}>Retry</Text>
          </TouchableOpacity>
        </View>
      </ScreenWrap>
    );
  }

  const summary = analytics?.summary;
  if (!summary || summary.totalQuizzes === 0) {
    return (
      <ScreenWrap>
        <ScrollView
          style={styles.screen}
          contentContainerStyle={skyScreen.content}
          refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => void loadAll(true)} tintColor={colors.brand} />}
        >
          <Text style={styles.pageTitle}>Performance</Text>
          <EmptyState
            title="No quiz data yet"
            message="Complete a quiz to unlock your analytics — trends, subjects, and weak topics will appear here."
          />
        </ScrollView>
      </ScreenWrap>
    );
  }

  const band = performanceBand(summary.averageScore);
  const accuracy = accuracyOf(summary.correctAnswers, summary.totalQuestionsAttempted);
  const weeks = (analytics?.weeklyScores ?? []).slice(-6);
  const days = last14Days(analytics?.recentActivity ?? []);
  const maxDayQuizzes = Math.max(1, ...days.map((d) => d.data?.quizCount ?? 0));
  const activeDays = days.filter((d) => (d.data?.quizCount ?? 0) > 0).length;
  const subjects = analytics?.subjectBreakdown ?? [];
  const weakest = topics.slice(0, 5);

  return (
    <ScreenWrap>
      <ScrollView
        style={styles.screen}
        contentContainerStyle={skyScreen.content}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => void loadAll(true)} tintColor={colors.brand} />}
      >
        <Text style={styles.pageTitle}>Performance</Text>
        <Text style={styles.pageLead}>A clear picture of how you're doing.</Text>

        {/* Hero mastery card */}
        <View style={styles.hero}>
          <ScoreRing score={summary.averageScore} />
          <View style={styles.heroRight}>
            <View style={[styles.bandPill, { backgroundColor: `${band.color}1A` }]}>
              <Ionicons name={band.icon} size={14} color={band.color} />
              <Text style={[styles.bandText, { color: band.color }]}>{band.label}</Text>
            </View>
            <Text style={styles.heroMetric}>{summary.averageScore.toFixed(0)}% average score</Text>
            <View style={styles.heroMetaRow}>
              <Ionicons name="documents-outline" size={14} color={colors.textMuted} />
              <Text style={styles.heroMeta}>
                {summary.totalQuizzes} quiz{summary.totalQuizzes === 1 ? "" : "zes"} · {accuracy.toFixed(0)}% accuracy
              </Text>
            </View>
            <View style={styles.heroMetaRow}>
              <Ionicons name="flame" size={14} color="#f97316" />
              <Text style={styles.heroMeta}>{summary.studyStreakDays}-day study streak</Text>
            </View>
          </View>
        </View>

        {/* Stat tiles */}
        <View style={styles.grid}>
          <StatTile
            icon="documents-outline"
            tint={colors.brand}
            label="Total Quizzes"
            value={String(summary.totalQuizzes)}
          />
          <StatTile
            icon="checkmark-done-outline"
            tint={scoreColor(accuracy)}
            label="Accuracy"
            value={`${accuracy.toFixed(0)}%`}
            sub={`${summary.correctAnswers}/${summary.totalQuestionsAttempted} correct`}
          />
          <StatTile
            icon="flame-outline"
            tint="#f97316"
            label="Study Streak"
            value={`${summary.studyStreakDays}d`}
          />
          <StatTile
            icon={summary.improvement >= 0 ? "trending-up-outline" : "trending-down-outline"}
            tint={summary.improvement >= 0 ? colors.success : colors.danger}
            label="vs Last Week"
            value={`${summary.improvement >= 0 ? "+" : ""}${summary.improvement.toFixed(1)}`}
            sub="points"
          />
        </View>

        {/* Insights */}
        {insights.length > 0 ? (
          <View style={styles.insightCard}>
            <View style={styles.insightHead}>
              <Ionicons name="sparkles" size={16} color={colors.brandDark} />
              <Text style={styles.insightHeadText}>Insights</Text>
            </View>
            {insights.map((ins, i) => (
              <View key={i} style={styles.insightRow}>
                <View style={[styles.insightDot, { backgroundColor: `${ins.color}1A` }]}>
                  <Ionicons name={ins.icon} size={14} color={ins.color} />
                </View>
                <Text style={styles.insightText}>{ins.text}</Text>
              </View>
            ))}
          </View>
        ) : null}

        {/* Score trend */}
        <SectionHeader icon="pulse-outline" title="Score Trend" caption="last 6 weeks" />
        {weeks.length >= 2 ? (
          <View style={styles.chartCard}>
            <LineChart
              data={{
                labels: weeks.map((w) => w.week.replace(/^\d+-/, "")),
                datasets: [{ data: weeks.map((w) => clampPct(w.score)) }],
              }}
              width={chartWidth}
              height={210}
              fromZero
              bezier
              yAxisSuffix="%"
              withInnerLines
              withOuterLines={false}
              chartConfig={lineChartConfig}
              style={styles.chart}
            />
          </View>
        ) : (
          <EmptyState small title="Not enough history" message="Practice across a few weeks to reveal your trend line." />
        )}

        {/* Activity strip */}
        <SectionHeader
          icon="calendar-outline"
          title="Activity"
          caption={`${activeDays}/14 active days`}
        />
        <View style={styles.card}>
          <View style={styles.activityRow}>
            {days.map((d) => {
              const count = d.data?.quizCount ?? 0;
              const h = 8 + Math.round((count / maxDayQuizzes) * 40);
              const filled = count > 0;
              return (
                <View key={d.key} style={styles.activityCol}>
                  <View
                    style={[
                      styles.activityBar,
                      {
                        height: h,
                        backgroundColor: filled ? scoreColor(d.data?.avgScore ?? 0) : colors.surfaceAlt,
                      },
                    ]}
                  />
                  <Text style={styles.activityDay}>{d.day}</Text>
                </View>
              );
            })}
          </View>
          <View style={styles.legendRow}>
            <LegendDot color={colors.success} label=">70%" />
            <LegendDot color={colors.warning} label="50–70%" />
            <LegendDot color={colors.danger} label="<50%" />
          </View>
        </View>

        {/* Subject performance */}
        <SectionHeader icon="library-outline" title="Subject Performance" />
        {subjects.length > 0 ? (
          <View style={styles.card}>
            {subjects.map((s: SubjectBreakdown, i) => {
              const tm = trendMeta(s.trend);
              return (
                <View key={s.subject} style={[styles.subjectRow, i === subjects.length - 1 ? styles.noBorder : null]}>
                  <View style={styles.subjectHeader}>
                    <Text style={styles.subjectName} numberOfLines={1}>
                      {s.subject}
                    </Text>
                    <Text style={[styles.subjectScore, { color: scoreColor(s.avgScore) }]}>
                      {s.avgScore.toFixed(0)}%
                    </Text>
                  </View>
                  <View style={styles.barTrack}>
                    <View
                      style={[
                        styles.barFill,
                        { width: `${clampPct(s.avgScore)}%`, backgroundColor: scoreColor(s.avgScore) },
                      ]}
                    />
                  </View>
                  <View style={styles.subjectMetaRow}>
                    <Text style={styles.subjectMeta}>
                      {s.quizCount} quiz{s.quizCount === 1 ? "" : "zes"}
                    </Text>
                    <View style={styles.trendChip}>
                      <Ionicons name={tm.icon} size={12} color={tm.color} />
                      <Text style={[styles.trendText, { color: tm.color }]}>{tm.label}</Text>
                    </View>
                  </View>
                </View>
              );
            })}
          </View>
        ) : (
          <EmptyState small title="No subjects yet" message="Complete quizzes to see subject performance." />
        )}

        {/* Weakest topics */}
        <SectionHeader icon="alert-circle-outline" title="Focus Areas" caption="weakest first" />
        {weakest.length > 0 ? (
          <View style={styles.cardList}>
            {weakest.map((t, i) => (
              <View key={`${t.subject}-${t.topic}`} style={styles.topicItem}>
                <View style={[styles.rankBadge, { backgroundColor: scoreColor(t.accuracy) }]}>
                  <Text style={styles.rankText}>{i + 1}</Text>
                </View>
                <View style={styles.recentLeft}>
                  <Text style={styles.recentTitle} numberOfLines={1}>
                    {t.topic}
                  </Text>
                  <Text style={styles.recentDate}>
                    {t.subject ? `${t.subject} · ` : ""}
                    {t.correctCount}/{t.totalAnswered} correct
                  </Text>
                </View>
                <View style={styles.topicRight}>
                  <Text style={[styles.accuracyValue, { color: scoreColor(t.accuracy) }]}>
                    {t.accuracy.toFixed(0)}%
                  </Text>
                  {t.sampleQuizId ? (
                    <TouchableOpacity
                      style={styles.practiceButton}
                      onPress={() => router.push(`/quiz/${t.sampleQuizId}`)}
                    >
                      <Ionicons name="play" size={12} color={colors.white} />
                      <Text style={styles.practiceText}>Practice</Text>
                    </TouchableOpacity>
                  ) : null}
                </View>
              </View>
            ))}
          </View>
        ) : (
          <EmptyState small title="No topic data yet" message="Answer more questions to reveal weak topics." />
        )}

        {/* Recent quizzes */}
        <SectionHeader icon="time-outline" title="Recent Quizzes" />
        {reports.length > 0 ? (
          <View style={styles.cardList}>
            {reports.map((item) => (
              <TouchableOpacity
                key={item.attemptId}
                style={styles.recentItem}
                onPress={() =>
                  router.push({
                    pathname: "/quiz/review",
                    params: { quizId: String(item.quizId), attemptId: String(item.attemptId) },
                  })
                }
              >
                <MiniRing score={item.score} />
                <View style={styles.recentLeft}>
                  <Text style={styles.recentTitle} numberOfLines={1}>
                    {item.quizTitle}
                  </Text>
                  <Text style={styles.recentDate}>
                    {item.completedAt ? new Date(item.completedAt).toLocaleDateString() : "—"}
                  </Text>
                </View>
                <Ionicons name="chevron-forward" size={18} color={colors.textSubtle} />
              </TouchableOpacity>
            ))}
          </View>
        ) : (
          <EmptyState small title="No recent quizzes" message="Your latest attempts show up here." />
        )}
      </ScrollView>
    </ScreenWrap>
  );
}

// --- components ---

function ScoreRing({ score, size = 116, stroke = 12 }: { score: number; size?: number; stroke?: number }) {
  const value = clampPct(score);
  const radius = (size - stroke) / 2;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (value / 100) * circumference;
  const color = scoreColor(value);
  return (
    <View style={{ width: size, height: size }}>
      <Svg width={size} height={size}>
        <G rotation="-90" origin={`${size / 2}, ${size / 2}`}>
          <Circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            stroke={colors.surfaceAlt}
            strokeWidth={stroke}
            fill="none"
          />
          <Circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            stroke={color}
            strokeWidth={stroke}
            strokeDasharray={circumference}
            strokeDashoffset={offset}
            strokeLinecap="round"
            fill="none"
          />
        </G>
      </Svg>
      <View style={styles.ringCenter}>
        <Text style={[styles.ringValue, { color }]}>{value.toFixed(0)}</Text>
        <Text style={styles.ringUnit}>avg %</Text>
      </View>
    </View>
  );
}

function MiniRing({ score, size = 46, stroke = 5 }: { score: number; size?: number; stroke?: number }) {
  const value = clampPct(score);
  const radius = (size - stroke) / 2;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (value / 100) * circumference;
  const color = scoreColor(value);
  return (
    <View style={{ width: size, height: size }}>
      <Svg width={size} height={size}>
        <G rotation="-90" origin={`${size / 2}, ${size / 2}`}>
          <Circle cx={size / 2} cy={size / 2} r={radius} stroke={colors.surfaceAlt} strokeWidth={stroke} fill="none" />
          <Circle
            cx={size / 2}
            cy={size / 2}
            r={radius}
            stroke={color}
            strokeWidth={stroke}
            strokeDasharray={circumference}
            strokeDashoffset={offset}
            strokeLinecap="round"
            fill="none"
          />
        </G>
      </Svg>
      <View style={styles.ringCenter}>
        <Text style={[styles.miniRingValue, { color }]}>{value.toFixed(0)}</Text>
      </View>
    </View>
  );
}

function StatTile({
  icon,
  tint,
  label,
  value,
  sub,
}: {
  icon: IoniconName;
  tint: string;
  label: string;
  value: string;
  sub?: string;
}) {
  return (
    <View style={styles.statCard}>
      <View style={[styles.statIcon, { backgroundColor: `${tint}1A` }]}>
        <Ionicons name={icon} size={18} color={tint} />
      </View>
      <Text style={styles.statLabel}>{label}</Text>
      <Text style={styles.statValue}>{value}</Text>
      {sub ? <Text style={styles.statSub}>{sub}</Text> : null}
    </View>
  );
}

function SectionHeader({ icon, title, caption }: { icon: IoniconName; title: string; caption?: string }) {
  return (
    <View style={styles.sectionHeader}>
      <View style={styles.sectionHeaderLeft}>
        <Ionicons name={icon} size={16} color={colors.brandDark} />
        <Text style={styles.sectionHeaderTitle}>{title}</Text>
      </View>
      {caption ? <Text style={styles.sectionCaption}>{caption}</Text> : null}
    </View>
  );
}

function LegendDot({ color, label }: { color: string; label: string }) {
  return (
    <View style={styles.legendItem}>
      <View style={[styles.legendSwatch, { backgroundColor: color }]} />
      <Text style={styles.legendText}>{label}</Text>
    </View>
  );
}

function EmptyState({ title, message, small }: { title: string; message: string; small?: boolean }) {
  return (
    <View style={[styles.empty, small ? styles.emptySmall : null]}>
      <View style={styles.emptyIconBadge}>
        <Ionicons name="bar-chart-outline" size={26} color={colors.brandDark} />
      </View>
      <Text style={styles.emptyTitle}>{title}</Text>
      <Text style={styles.emptyMessage}>{message}</Text>
    </View>
  );
}

const lineChartConfig = {
  backgroundGradientFrom: "#ffffff",
  backgroundGradientTo: "#ffffff",
  decimalPlaces: 0,
  color: (opacity = 1) => `rgba(32, 178, 170, ${opacity})`,
  labelColor: () => colors.textMuted,
  propsForBackgroundLines: { stroke: "#eef0f6" },
  propsForDots: { r: "5", strokeWidth: "2", stroke: "#20B2AA", fill: "#ffffff" },
  fillShadowGradient: "#20B2AA",
  fillShadowGradientOpacity: 0.18,
};

const glass = skyScreen.glass;

const styles = StyleSheet.create({
  screen: { flex: 1 },
  muted: { color: colors.textMuted },
  error: { color: colors.danger, textAlign: "center" },

  pageTitle: {
    fontSize: 28,
    fontWeight: "800",
    color: "#FFD400",
    letterSpacing: -0.4,
    textAlign: "center",
    textShadowColor: "rgba(15, 23, 42, 0.35)",
    textShadowOffset: { width: 0, height: 1 },
    textShadowRadius: 4,
  },
  pageLead: {
    color: "#F97316",
    fontSize: 14,
    lineHeight: 20,
    fontWeight: "700",
    textAlign: "center",
  },

  hero: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.lg,
    borderWidth: 1,
    borderRadius: radius.xl,
    padding: spacing.lg,
    ...glass,
    ...shadow.md,
  },
  heroRight: { flex: 1, gap: 6 },
  bandPill: {
    flexDirection: "row",
    alignItems: "center",
    gap: 5,
    alignSelf: "flex-start",
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: radius.full,
  },
  bandText: { fontSize: 12, fontWeight: "800" },
  heroMetric: { fontSize: 16, fontWeight: "800", color: colors.text, marginTop: 2 },
  heroMetaRow: { flexDirection: "row", alignItems: "center", gap: 6 },
  heroMeta: { color: colors.textMuted, fontSize: 13, fontWeight: "600" },

  ringCenter: {
    position: "absolute",
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    alignItems: "center",
    justifyContent: "center",
  },
  ringValue: { fontSize: 30, fontWeight: "900", lineHeight: 34 },
  ringUnit: { fontSize: 11, color: colors.textMuted, fontWeight: "700", marginTop: 2 },
  miniRingValue: { fontSize: 13, fontWeight: "800" },

  grid: { flexDirection: "row", flexWrap: "wrap", gap: 12 },
  statCard: {
    width: (screenWidth - CONTENT_PAD * 2 - 12) / 2,
    borderWidth: 1,
    borderRadius: radius.lg,
    padding: 16,
    gap: 4,
    ...glass,
    ...shadow.sm,
  },
  statIcon: {
    width: 34,
    height: 34,
    borderRadius: 10,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: 6,
  },
  statLabel: { color: colors.textMuted, fontSize: 13, fontWeight: "600" },
  statValue: { fontSize: 24, fontWeight: "800", color: colors.text },
  statSub: { color: colors.textSubtle, fontSize: 12, fontWeight: "600" },

  insightCard: {
    borderWidth: 1,
    borderRadius: radius.lg,
    padding: spacing.lg,
    gap: 12,
    ...glass,
    ...shadow.sm,
  },
  insightHead: { flexDirection: "row", alignItems: "center", gap: 7 },
  insightHeadText: { fontWeight: "800", color: colors.text, fontSize: 15 },
  insightRow: { flexDirection: "row", alignItems: "center", gap: 11 },
  insightDot: { width: 30, height: 30, borderRadius: 9, alignItems: "center", justifyContent: "center" },
  insightText: { flex: 1, color: colors.text, fontSize: 13.5, lineHeight: 19 },

  sectionHeader: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    marginTop: 4,
    marginBottom: 2,
  },
  sectionHeaderLeft: { flexDirection: "row", alignItems: "center", gap: 7 },
  sectionHeaderTitle: { fontSize: 16, fontWeight: "800", color: colors.text },
  sectionCaption: { fontSize: 12, color: colors.textMuted, fontWeight: "600" },

  chartCard: {
    borderRadius: radius.lg,
    borderWidth: 1,
    paddingVertical: 8,
    paddingRight: 8,
    alignItems: "center",
    overflow: "hidden",
    ...glass,
    ...shadow.sm,
  },
  chart: { borderRadius: radius.lg, marginVertical: 4 },

  card: { borderWidth: 1, borderRadius: radius.lg, padding: 16, ...glass, ...shadow.sm },
  cardList: { gap: 10 },

  activityRow: { flexDirection: "row", alignItems: "flex-end", justifyContent: "space-between", height: 60 },
  activityCol: { alignItems: "center", flex: 1, gap: 4 },
  activityBar: { width: 8, borderRadius: radius.full },
  activityDay: { fontSize: 9, color: colors.textSubtle, fontWeight: "600" },
  legendRow: { flexDirection: "row", justifyContent: "center", gap: 16, marginTop: 12 },
  legendItem: { flexDirection: "row", alignItems: "center", gap: 5 },
  legendSwatch: { width: 10, height: 10, borderRadius: 3 },
  legendText: { fontSize: 11, color: colors.textMuted, fontWeight: "600" },

  subjectRow: { marginBottom: 14, paddingBottom: 14, borderBottomWidth: 1, borderBottomColor: colors.border },
  noBorder: { marginBottom: 0, paddingBottom: 0, borderBottomWidth: 0 },
  subjectHeader: { flexDirection: "row", justifyContent: "space-between", marginBottom: 7 },
  subjectName: { fontWeight: "700", color: colors.text, flex: 1, paddingRight: 10 },
  subjectScore: { fontWeight: "800" },
  barTrack: { height: 10, borderRadius: radius.full, backgroundColor: colors.surfaceAlt, overflow: "hidden" },
  barFill: { height: 10, borderRadius: radius.full },
  subjectMetaRow: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", marginTop: 6 },
  subjectMeta: { color: colors.textMuted, fontSize: 12, fontWeight: "600" },
  trendChip: { flexDirection: "row", alignItems: "center", gap: 4 },
  trendText: { fontSize: 12, fontWeight: "700" },

  recentItem: {
    flexDirection: "row",
    alignItems: "center",
    gap: 12,
    borderWidth: 1,
    borderRadius: radius.md,
    padding: 12,
    ...glass,
    ...shadow.sm,
  },
  recentLeft: { flex: 1, paddingRight: 8 },
  recentTitle: { fontWeight: "700", color: colors.text, marginBottom: 2 },
  recentDate: { color: colors.textMuted, fontSize: 13 },

  topicItem: {
    flexDirection: "row",
    alignItems: "center",
    gap: 12,
    borderWidth: 1,
    borderRadius: radius.md,
    padding: 12,
    ...glass,
    ...shadow.sm,
  },
  rankBadge: { width: 26, height: 26, borderRadius: radius.full, alignItems: "center", justifyContent: "center" },
  rankText: { color: colors.white, fontWeight: "800", fontSize: 13 },
  topicRight: { flexDirection: "row", alignItems: "center", gap: 10 },
  accuracyValue: { fontWeight: "800", fontSize: 15 },
  practiceButton: {
    flexDirection: "row",
    alignItems: "center",
    gap: 4,
    backgroundColor: colors.brand,
    borderRadius: radius.sm,
    paddingHorizontal: 10,
    paddingVertical: 6,
    ...shadow.brand,
  },
  practiceText: { color: colors.white, fontWeight: "700", fontSize: 12 },

  empty: {
    alignItems: "center",
    borderWidth: 1,
    borderStyle: "dashed",
    borderRadius: radius.lg,
    padding: 40,
    ...glass,
  },
  emptySmall: { padding: 20 },
  emptyIconBadge: {
    width: 52,
    height: 52,
    borderRadius: radius.full,
    backgroundColor: colors.brandSoft,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: 10,
  },
  emptyTitle: { fontWeight: "700", color: colors.text, marginBottom: 4 },
  emptyMessage: { color: colors.textMuted, textAlign: "center", fontSize: 14 },
  retryButton: { backgroundColor: colors.brand, borderRadius: radius.md, paddingVertical: 11, paddingHorizontal: 18, ...shadow.brand },
  retryText: { color: colors.white, fontWeight: "700" },
});
