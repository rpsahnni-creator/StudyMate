import React, { useCallback, useState } from "react";
import {
  ActivityIndicator,
  Dimensions,
  FlatList,
  ScrollView,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import { router, useFocusEffect } from "expo-router";
import { Ionicons } from "@expo/vector-icons";
import { BarChart } from "react-native-chart-kit";
import { SkyBackground } from "../../components/SkyBackground";
import {
  getAnalytics,
  getMyReports,
  getTopicAnalytics,
  type Analytics,
  type ReportItem,
  type TopicAccuracy,
} from "../../lib/api";
import { skyScreen } from "../../lib/skyScreen";
import { colors, radius, shadow, scoreColor } from "../../lib/theme";

const RECENT_LIMIT = 8;
const screenWidth = Dimensions.get("window").width;

function trendLabel(trend: string): string {
  switch (trend) {
    case "improving":
      return "▲ Improving";
    case "declining":
      return "▼ Declining";
    default:
      return "▬ Stable";
  }
}

const chartConfig = {
  backgroundGradientFrom: "#ffffff",
  backgroundGradientTo: "#ffffff",
  decimalPlaces: 0,
  color: (opacity = 1) => `rgba(99, 102, 241, ${opacity})`,
  labelColor: () => colors.textMuted,
  barPercentage: 0.6,
  propsForBackgroundLines: { stroke: "#eef0f6" },
  fillShadowGradient: "#6366f1",
  fillShadowGradientOpacity: 1,
};

function ScreenWrap({ children }: { children: React.ReactNode }) {
  return <SkyBackground>{children}</SkyBackground>;
}

export default function ReportsScreen() {
  const [analytics, setAnalytics] = useState<Analytics | null>(null);
  const [reports, setReports] = useState<ReportItem[]>([]);
  const [topics, setTopics] = useState<TopicAccuracy[]>([]);
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
      setReports(r.reports);
      setTopics(t.topics);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load analytics");
    } finally {
      setLoading(false);
    }
  }, []);

  useFocusEffect(
    useCallback(() => {
      void loadAll();
    }, [loadAll])
  );

  if (loading) {
    return (
      <ScreenWrap>
        <View style={skyScreen.center}>
          <ActivityIndicator size="large" color={colors.brand} />
          <Text style={styles.muted}>Loading analytics…</Text>
        </View>
      </ScreenWrap>
    );
  }

  if (error) {
    return (
      <ScreenWrap>
        <View style={skyScreen.center}>
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
        <ScrollView style={styles.screen} contentContainerStyle={skyScreen.content}>
          <Text style={skyScreen.title}>Performance</Text>
          <EmptyState
            title="No quiz data yet"
            message="Complete a quiz to unlock your analytics — trends, subjects, and weak topics will appear here."
          />
        </ScrollView>
      </ScreenWrap>
    );
  }

  const weeks = (analytics?.weeklyScores ?? []).slice(-5);
  const weakest = topics.slice(0, 5);
  const maxSubjectScore = 100;

  return (
    <ScreenWrap>
      <ScrollView style={styles.screen} contentContainerStyle={skyScreen.content}>
        <Text style={skyScreen.title}>Performance</Text>
        <Text style={skyScreen.lead}>Your progress at a glance.</Text>

        <View style={styles.grid}>
          <StatCard label="Total Quizzes" value={String(summary.totalQuizzes)} />
          <StatCard
            label="Avg Score"
            value={`${summary.averageScore.toFixed(0)}%`}
            color={scoreColor(summary.averageScore)}
          />
          <StatCard label="Study Streak" value={`${summary.studyStreakDays}d`} />
          <StatCard
            label="This vs Last Wk"
            value={`${summary.improvement >= 0 ? "+" : ""}${summary.improvement.toFixed(1)}`}
            color={summary.improvement >= 0 ? "#16a34a" : "#dc2626"}
          />
        </View>

        <Text style={skyScreen.sectionTitle}>Score Trend (last 5 weeks)</Text>
        {weeks.length > 0 ? (
          <View style={styles.chartWrap}>
            <BarChart
              data={{
                labels: weeks.map((w) => w.week.replace(/^\d+-/, "")),
                datasets: [{ data: weeks.map((w) => w.score) }],
              }}
              width={screenWidth - 40}
              height={220}
              fromZero
              yAxisLabel=""
              yAxisSuffix="%"
              chartConfig={chartConfig}
              style={styles.chart}
            />
          </View>
        ) : (
          <EmptyState small title="Not enough history" message="Practice across weeks to see a trend." />
        )}

        <Text style={skyScreen.sectionTitle}>Subject Performance</Text>
        {analytics && analytics.subjectBreakdown.length > 0 ? (
          <View style={styles.card}>
            {analytics.subjectBreakdown.map((s) => (
              <View key={s.subject} style={styles.subjectRow}>
                <View style={styles.subjectHeader}>
                  <Text style={styles.subjectName}>{s.subject}</Text>
                  <Text style={[styles.subjectScore, { color: scoreColor(s.avgScore) }]}>
                    {s.avgScore.toFixed(0)}%
                  </Text>
                </View>
                <View style={styles.barTrack}>
                  <View
                    style={[
                      styles.barFill,
                      { width: `${(s.avgScore / maxSubjectScore) * 100}%`, backgroundColor: scoreColor(s.avgScore) },
                    ]}
                  />
                </View>
                <Text style={styles.subjectMeta}>
                  {s.quizCount} quiz{s.quizCount === 1 ? "" : "zes"} · {trendLabel(s.trend)}
                </Text>
              </View>
            ))}
          </View>
        ) : (
          <EmptyState small title="No subjects yet" message="Complete quizzes to see subject performance." />
        )}

        <Text style={skyScreen.sectionTitle}>Recent Quizzes</Text>
        {reports.length > 0 ? (
          <FlatList
            data={reports}
            scrollEnabled={false}
            keyExtractor={(item: ReportItem) => String(item.attemptId)}
            contentContainerStyle={styles.cardList}
            renderItem={({ item }: { item: ReportItem }) => (
              <TouchableOpacity
                style={styles.recentItem}
                onPress={() =>
                  router.push({
                    pathname: "/quiz/review",
                    params: { quizId: String(item.quizId), attemptId: String(item.attemptId) },
                  })
                }
              >
                <View style={styles.recentLeft}>
                  <Text style={styles.recentTitle} numberOfLines={1}>
                    {item.quizTitle}
                  </Text>
                  <Text style={styles.recentDate}>
                    {item.completedAt ? new Date(item.completedAt).toLocaleDateString() : "—"}
                  </Text>
                </View>
                <Text style={[styles.badge, { backgroundColor: scoreColor(item.score) }]}>
                  {item.score.toFixed(0)}%
                </Text>
              </TouchableOpacity>
            )}
          />
        ) : (
          <EmptyState small title="No recent quizzes" message="Your latest attempts show up here." />
        )}

        <Text style={skyScreen.sectionTitle}>Weakest Topics</Text>
        {weakest.length > 0 ? (
          <View style={styles.cardList}>
            {weakest.map((t) => (
              <View key={`${t.subject}-${t.topic}`} style={styles.topicItem}>
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
                  <Text style={[styles.badge, { backgroundColor: scoreColor(t.accuracy) }]}>
                    {t.accuracy.toFixed(0)}%
                  </Text>
                  {t.sampleQuizId ? (
                    <TouchableOpacity
                      style={styles.practiceButton}
                      onPress={() => router.push(`/quiz/${t.sampleQuizId}`)}
                    >
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
      </ScrollView>
    </ScreenWrap>
  );
}

function StatCard({ label, value, color }: { label: string; value: string; color?: string }) {
  return (
    <View style={styles.statCard}>
      <Text style={styles.statLabel}>{label}</Text>
      <Text style={[styles.statValue, color ? { color } : null]}>{value}</Text>
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

const glass = skyScreen.glass;

const styles = StyleSheet.create({
  screen: { flex: 1 },
  muted: { color: colors.textMuted },
  error: { color: colors.danger },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: 12 },
  statCard: {
    width: (screenWidth - 40 - 12) / 2,
    borderWidth: 1,
    borderRadius: radius.lg,
    padding: 16,
    ...glass,
    ...shadow.sm,
  },
  statLabel: { color: colors.textMuted, fontSize: 13, marginBottom: 6, fontWeight: "600" },
  statValue: { fontSize: 24, fontWeight: "800", color: colors.text },
  chartWrap: {
    borderRadius: radius.lg,
    overflow: "hidden",
    ...glass,
    borderWidth: 1,
    ...shadow.sm,
  },
  chart: { borderRadius: radius.lg, marginVertical: 4 },
  card: { borderWidth: 1, borderRadius: radius.lg, padding: 16, ...glass, ...shadow.sm },
  cardList: { gap: 10 },
  subjectRow: { marginBottom: 14 },
  subjectHeader: { flexDirection: "row", justifyContent: "space-between", marginBottom: 6 },
  subjectName: { fontWeight: "700", color: colors.text },
  subjectScore: { fontWeight: "800" },
  barTrack: { height: 10, borderRadius: radius.full, backgroundColor: colors.surfaceAlt, overflow: "hidden" },
  barFill: { height: 10, borderRadius: radius.full },
  subjectMeta: { color: colors.textMuted, fontSize: 12, marginTop: 4 },
  recentItem: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    borderWidth: 1,
    borderRadius: radius.md,
    padding: 14,
    ...glass,
    ...shadow.sm,
  },
  recentLeft: { flex: 1, paddingRight: 12 },
  recentTitle: { fontWeight: "700", color: colors.text, marginBottom: 2 },
  recentDate: { color: colors.textMuted, fontSize: 13 },
  badge: {
    color: colors.white,
    fontWeight: "800",
    paddingHorizontal: 10,
    paddingVertical: 5,
    borderRadius: radius.full,
    fontSize: 13,
    overflow: "hidden",
  },
  topicItem: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    borderWidth: 1,
    borderRadius: radius.md,
    padding: 14,
    ...glass,
    ...shadow.sm,
  },
  topicRight: { flexDirection: "row", alignItems: "center", gap: 10 },
  practiceButton: { backgroundColor: colors.brand, borderRadius: radius.sm, paddingHorizontal: 12, paddingVertical: 6, ...shadow.brand },
  practiceText: { color: colors.white, fontWeight: "700", fontSize: 13 },
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
