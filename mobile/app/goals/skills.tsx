import React from "react";
import {
  ActivityIndicator,
  ScrollView,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import { router } from "expo-router";
import { useSkillGaps } from "../../hooks/useCareerGoals";
import type { SkillGap } from "../../lib/careerGoals";
import { colors, radius, shadow } from "../../lib/theme";

function colorFor(weakness: number): string {
  if (weakness > 70) return colors.danger; // red — weak
  if (weakness >= 30) return colors.warning; // yellow — improving
  return colors.success; // green — strong
}

function labelFor(weakness: number): string {
  if (weakness > 70) return "Needs work";
  if (weakness >= 30) return "Improving";
  return "Strong";
}

export default function SkillsScreen() {
  const skills = useSkillGaps();

  if (skills.gated) {
    return (
      <View style={styles.center}>
        <Text style={styles.title}>Coming Soon</Text>
        <Text style={styles.muted}>Skill tracking isn't available yet.</Text>
      </View>
    );
  }

  if (skills.loading && !skills.data) {
    return (
      <View style={styles.center}>
        <ActivityIndicator size="large" />
        <Text style={styles.muted}>Loading your skills…</Text>
      </View>
    );
  }

  if (skills.error) {
    return (
      <View style={styles.center}>
        <Text style={styles.error}>{skills.error}</Text>
        <TouchableOpacity style={styles.primaryButton} onPress={skills.reload}>
          <Text style={styles.primaryButtonText}>Retry</Text>
        </TouchableOpacity>
      </View>
    );
  }

  const list = skills.data ?? [];

  if (list.length === 0) {
    return (
      <View style={styles.center}>
        <Text style={styles.title}>No skills tracked yet</Text>
        <Text style={styles.muted}>Complete a daily practice to start mapping your strengths.</Text>
        <TouchableOpacity style={styles.primaryButton} onPress={() => router.replace("/goals/practice")}>
          <Text style={styles.primaryButtonText}>Start Practice</Text>
        </TouchableOpacity>
      </View>
    );
  }

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <Text style={styles.title}>Your Skills</Text>
      <Text style={styles.muted}>Weakest topics first — focus here for the biggest gains.</Text>
      {list.map((gap) => (
        <SkillRow key={`${gap.subject}-${gap.topic}`} gap={gap} />
      ))}
    </ScrollView>
  );
}

function SkillRow({ gap }: { gap: SkillGap }) {
  const color = colorFor(gap.weaknessScore);
  const width = Math.max(4, Math.min(100, gap.weaknessScore));

  return (
    <View style={styles.card}>
      <View style={styles.rowHeader}>
        <View style={styles.rowTitleWrap}>
          <Text style={styles.topic}>{gap.topic}</Text>
          <Text style={styles.subject}>{gap.subject}</Text>
        </View>
        <Text style={[styles.status, { color }]}>{labelFor(gap.weaknessScore)}</Text>
      </View>

      <View style={styles.barTrack}>
        <View style={[styles.barFill, { width: `${width}%`, backgroundColor: color }]} />
      </View>

      <View style={styles.rowFooter}>
        <Text style={styles.weakness}>Needs work: {gap.weaknessScore.toFixed(0)}%</Text>
        <TouchableOpacity onPress={() => router.replace("/goals/practice")}>
          <Text style={styles.focusLink}>Focus Topic →</Text>
        </TouchableOpacity>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.bg },
  content: { padding: 20, gap: 12 },
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: 12, padding: 24, backgroundColor: colors.bg },
  title: { fontSize: 23, fontWeight: "800", color: colors.text },
  muted: { color: colors.textMuted, textAlign: "center", lineHeight: 20 },
  error: { color: colors.danger },
  card: {
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.lg,
    padding: 16,
    gap: 10,
    backgroundColor: colors.surface,
    ...shadow.sm,
  },
  rowHeader: { flexDirection: "row", justifyContent: "space-between", alignItems: "flex-start" },
  rowTitleWrap: { flex: 1, gap: 2 },
  topic: { fontSize: 16, fontWeight: "800", color: colors.text },
  subject: { fontSize: 13, color: colors.textMuted },
  status: { fontSize: 13, fontWeight: "800" },
  barTrack: { height: 10, borderRadius: radius.full, backgroundColor: colors.surfaceAlt, overflow: "hidden" },
  barFill: { height: "100%", borderRadius: radius.full },
  rowFooter: { flexDirection: "row", justifyContent: "space-between", alignItems: "center" },
  weakness: { fontSize: 13, color: colors.textMuted },
  focusLink: { color: colors.brandDark, fontWeight: "700" },
  primaryButton: {
    backgroundColor: colors.brand,
    borderRadius: radius.md,
    paddingVertical: 13,
    paddingHorizontal: 20,
    alignItems: "center",
    ...shadow.brand,
  },
  primaryButtonText: { color: colors.white, fontWeight: "700", fontSize: 16 },
});
