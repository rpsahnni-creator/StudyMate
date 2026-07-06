import React from "react";
import { StyleSheet, Text, TouchableOpacity, View } from "react-native";
import { router, useLocalSearchParams } from "expo-router";
import { colors, radius, shadow, scoreColor } from "../../lib/theme";

function formatDuration(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  if (m <= 0) return `${s}s`;
  return `${m}m ${s}s`;
}

export default function QuizResultScreen() {
  const params = useLocalSearchParams<{
    quizId: string;
    attemptId: string;
    score: string;
    correct: string;
    wrong: string;
    skipped: string;
    total: string;
    timeTaken: string;
  }>();

  const score = Number(params.score ?? 0);
  const correct = Number(params.correct ?? 0);
  const wrong = Number(params.wrong ?? 0);
  const skipped = Number(params.skipped ?? 0);
  const total = Number(params.total ?? 0);
  const timeTaken = Number(params.timeTaken ?? 0);

  const sc = scoreColor(score);
  const passing = score >= 60;

  return (
    <View style={styles.container}>
      <Text style={styles.emoji}>{passing ? "🎉" : "💪"}</Text>
      <Text style={styles.title}>{passing ? "Well done!" : "Keep going!"}</Text>
      <View style={[styles.scoreCircle, { borderColor: sc, backgroundColor: `${sc}14` }]}>
        <Text style={[styles.scoreText, { color: sc }]}>{score.toFixed(1)}%</Text>
      </View>

      <View style={styles.statsRow}>
        <Stat label="Correct" value={correct} color={colors.success} />
        <Stat label="Wrong" value={wrong} color={colors.danger} />
        <Stat label="Skipped" value={skipped} color={colors.warning} />
      </View>

      <Text style={styles.meta}>
        {total} questions · Time {formatDuration(timeTaken)}
      </Text>

      <View style={styles.buttonRow}>
        <TouchableOpacity
          style={styles.primaryButton}
          onPress={() =>
            router.replace({
              pathname: "/quiz/review",
              params: { quizId: String(params.quizId), attemptId: String(params.attemptId) },
            })
          }
        >
          <Text style={styles.primaryButtonText}>Review Answers</Text>
        </TouchableOpacity>
        <TouchableOpacity style={styles.secondaryButton} onPress={() => router.replace("/(tabs)")}>
          <Text style={styles.secondaryButtonText}>Home</Text>
        </TouchableOpacity>
      </View>
    </View>
  );
}

function Stat({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <View style={styles.stat}>
      <Text style={[styles.statValue, { color }]}>{value}</Text>
      <Text style={styles.statLabel}>{label}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, padding: 24, alignItems: "center", justifyContent: "center", gap: 18, backgroundColor: colors.bg },
  emoji: { fontSize: 46 },
  title: { fontSize: 24, fontWeight: "800", color: colors.text },
  scoreCircle: {
    width: 170,
    height: 170,
    borderRadius: 999,
    borderWidth: 7,
    alignItems: "center",
    justifyContent: "center",
  },
  scoreText: { fontSize: 38, fontWeight: "800" },
  statsRow: { flexDirection: "row", gap: 32 },
  stat: { alignItems: "center" },
  statValue: { fontSize: 30, fontWeight: "800" },
  statLabel: { color: colors.textMuted, fontSize: 13, fontWeight: "600" },
  meta: { color: colors.textMuted },
  buttonRow: { flexDirection: "row", gap: 12, marginTop: 8 },
  primaryButton: { backgroundColor: colors.brand, borderRadius: radius.md, paddingVertical: 13, paddingHorizontal: 20, ...shadow.brand },
  primaryButtonText: { color: colors.white, fontWeight: "700", fontSize: 16 },
  secondaryButton: {
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.borderStrong,
    borderRadius: radius.md,
    paddingVertical: 13,
    paddingHorizontal: 20,
  },
  secondaryButtonText: { color: colors.text, fontWeight: "700", fontSize: 16 },
});
