import React, { useCallback } from "react";
import {
  ActivityIndicator,
  ScrollView,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import { router, useFocusEffect } from "expo-router";
import { Ionicons } from "@expo/vector-icons";
import { useGoals, useMyGoal } from "../../hooks/useCareerGoals";
import type { CareerGoal, MyGoal } from "../../lib/careerGoals";
import { colors, radius, shadow, spacing } from "../../lib/theme";

export default function GoalsScreen() {
  const myGoal = useMyGoal();
  const goals = useGoals();

  useFocusEffect(
    useCallback(() => {
      myGoal.reload();
      // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [])
  );

  // The backend owns flag gating; a 403 surfaces as `gated`.
  if (myGoal.gated || goals.gated) {
    return <ComingSoon />;
  }

  if (myGoal.loading && !myGoal.data) {
    return (
      <View style={styles.center}>
        <ActivityIndicator size="large" />
        <Text style={styles.muted}>Loading your goal…</Text>
      </View>
    );
  }

  if (myGoal.error) {
    return (
      <View style={styles.center}>
        <Text style={styles.error}>{myGoal.error}</Text>
        <TouchableOpacity style={styles.primaryButton} onPress={myGoal.reload}>
          <Text style={styles.primaryButtonText}>Retry</Text>
        </TouchableOpacity>
      </View>
    );
  }

  if (myGoal.data) {
    return <ActiveGoalView goal={myGoal.data} />;
  }

  return <ChooseGoalView goals={goals} />;
}

function ComingSoon() {
  return (
    <View style={styles.center}>
      <View style={styles.comingSoonIcon}>
        <Ionicons name="flag" size={30} color={colors.brandDark} />
      </View>
      <Text style={styles.title}>Coming Soon</Text>
      <Text style={[styles.muted, styles.comingSoonText]}>
        Career goals and daily practice are on the way. Check back shortly!
      </Text>
    </View>
  );
}

function ChooseGoalView({ goals }: { goals: ReturnType<typeof useGoals> }) {
  if (goals.loading && !goals.data) {
    return (
      <View style={styles.center}>
        <ActivityIndicator size="large" />
        <Text style={styles.muted}>Loading goals…</Text>
      </View>
    );
  }

  const list = goals.data ?? [];

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <Text style={styles.hero}>Choose Your Goal</Text>
      <Text style={styles.muted}>
        Pick an exam to unlock a personalized daily practice plan.
      </Text>

      {goals.error ? <Text style={styles.error}>{goals.error}</Text> : null}

      {list.map((goal) => (
        <GoalCard key={goal.id} goal={goal} />
      ))}

      {list.length === 0 && !goals.loading ? (
        <Text style={styles.muted}>No goals available yet.</Text>
      ) : null}
    </ScrollView>
  );
}

function GoalCard({ goal }: { goal: CareerGoal }) {
  return (
    <View style={styles.card}>
      <Text style={styles.cardTitle}>{goal.name}</Text>
      <Text style={styles.badge}>{goal.examName || "Exam"}</Text>
      <Text style={styles.muted}>{goal.description}</Text>
      <View style={styles.metaRow}>
        <Text style={styles.metaChip}>{goal.targetMonths} months</Text>
        <Text style={styles.metaChip}>{goal.subjectAreas.length} subjects</Text>
      </View>
      <TouchableOpacity
        style={styles.primaryButton}
        onPress={() =>
          router.push({ pathname: "/goals/select", params: { goalId: String(goal.id) } })
        }
      >
        <Text style={styles.primaryButtonText}>Select</Text>
      </TouchableOpacity>
    </View>
  );
}

function ActiveGoalView({ goal }: { goal: MyGoal }) {
  const { progress } = goal;
  const days = goal.daysRemaining;

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <View style={styles.goalHeader}>
        <Text style={styles.title}>{goal.name}</Text>
        <Text style={styles.badge}>{goal.examName || "Exam"}</Text>
        {goal.targetDate ? (
          <Text style={styles.muted}>Target date: {goal.targetDate}</Text>
        ) : null}
      </View>

      <View style={styles.ringRow}>
        <View style={styles.ring}>
          <Text style={styles.ringValue}>{days ?? "—"}</Text>
          <Text style={styles.ringLabel}>days left</Text>
        </View>
        <View style={styles.streakBox}>
          <Text style={styles.streakValue}>🔥 {progress.currentStreak}</Text>
          <Text style={styles.ringLabel}>day streak</Text>
        </View>
      </View>

      <TodayCard progress={progress} />

      <TouchableOpacity
        style={styles.secondaryButton}
        onPress={() => router.push("/goals/skills")}
      >
        <Text style={styles.secondaryButtonText}>View Skills</Text>
      </TouchableOpacity>

      {progress.averageScore != null ? (
        <Text style={styles.muted}>
          Average score: {progress.averageScore.toFixed(1)}% over {progress.completedPractices}{" "}
          practices
        </Text>
      ) : null}
    </ScrollView>
  );
}

function TodayCard({ progress }: { progress: MyGoal["progress"] }) {
  const completed = progress.todayStatus === "completed";

  return (
    <TouchableOpacity
      style={[styles.todayCard, completed ? styles.todayDone : styles.todayPending]}
      onPress={() => router.push("/goals/practice")}
      disabled={false}
    >
      <Text style={styles.todayLabel}>Today's Practice</Text>
      {completed ? (
        <Text style={styles.todayValue}>
          Completed: {progress.todayScore != null ? `${progress.todayScore.toFixed(0)}%` : "done"}
        </Text>
      ) : (
        <Text style={styles.todayValue}>Practice Pending — tap to start</Text>
      )}
    </TouchableOpacity>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.bg },
  content: { padding: 20, gap: 14 },
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: 12, padding: 24, backgroundColor: colors.bg },
  comingSoonIcon: {
    width: 64,
    height: 64,
    borderRadius: radius.full,
    backgroundColor: colors.brandSoft,
    alignItems: "center",
    justifyContent: "center",
    marginBottom: spacing.xs,
  },
  comingSoonText: { textAlign: "center", maxWidth: 280 },
  hero: { fontSize: 26, fontWeight: "800", color: colors.text },
  title: { fontSize: 23, fontWeight: "800", color: colors.text },
  muted: { color: colors.textMuted, lineHeight: 20 },
  error: { color: colors.danger },
  card: {
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.lg,
    padding: 18,
    gap: 8,
    backgroundColor: colors.surface,
    ...shadow.sm,
  },
  cardTitle: { fontSize: 18, fontWeight: "800", color: colors.text },
  badge: {
    alignSelf: "flex-start",
    backgroundColor: colors.brandSoft,
    color: colors.brandDark,
    paddingHorizontal: 11,
    paddingVertical: 4,
    borderRadius: 999,
    fontSize: 13,
    fontWeight: "700",
    overflow: "hidden",
  },
  metaRow: { flexDirection: "row", gap: 8 },
  metaChip: {
    backgroundColor: colors.surfaceAlt,
    color: colors.textMuted,
    paddingHorizontal: 11,
    paddingVertical: 5,
    borderRadius: radius.sm,
    fontSize: 13,
    fontWeight: "700",
    overflow: "hidden",
  },
  goalHeader: { gap: 6 },
  ringRow: { flexDirection: "row", gap: 16, alignItems: "center" },
  ring: {
    width: 122,
    height: 122,
    borderRadius: 999,
    borderWidth: 8,
    borderColor: colors.brand,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.brandSoft,
  },
  ringValue: { fontSize: 34, fontWeight: "800", color: colors.brandDark },
  ringLabel: { color: colors.textMuted, fontSize: 13 },
  streakBox: {
    flex: 1,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.lg,
    padding: 16,
    alignItems: "center",
    gap: 4,
    backgroundColor: colors.surface,
    ...shadow.sm,
  },
  streakValue: { fontSize: 26, fontWeight: "800", color: "#ea580c" },
  todayCard: { borderRadius: radius.lg, padding: 18, gap: 6 },
  todayPending: { backgroundColor: colors.warningBg, borderWidth: 1, borderColor: "#fcd34d" },
  todayDone: { backgroundColor: colors.successBg, borderWidth: 1, borderColor: "#86efac" },
  todayLabel: { fontSize: 14, fontWeight: "700", color: colors.textMuted },
  todayValue: { fontSize: 18, fontWeight: "800", color: colors.text },
  primaryButton: {
    backgroundColor: colors.brand,
    borderRadius: radius.md,
    paddingVertical: 13,
    paddingHorizontal: 20,
    alignItems: "center",
    ...shadow.brand,
  },
  primaryButtonText: { color: colors.white, fontWeight: "700", fontSize: 16 },
  secondaryButton: {
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.borderStrong,
    borderRadius: radius.md,
    paddingVertical: 13,
    paddingHorizontal: 20,
    alignItems: "center",
  },
  secondaryButtonText: { color: colors.text, fontWeight: "700", fontSize: 16 },
});
