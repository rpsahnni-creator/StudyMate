import React, { useMemo, useState } from "react";
import {
  ActivityIndicator,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from "react-native";
import { router, useLocalSearchParams } from "expo-router";
import { useGoals } from "../../hooks/useCareerGoals";
import { FeatureGatedError, selectGoal } from "../../lib/careerGoals";
import { colors, radius } from "../../lib/theme";

function addMonths(months: number): string {
  const d = new Date();
  d.setMonth(d.getMonth() + months);
  return d.toISOString().slice(0, 10);
}

function isValidDate(value: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) return false;
  const t = Date.parse(value);
  return !Number.isNaN(t);
}

export default function SelectGoalScreen() {
  const params = useLocalSearchParams<{ goalId: string }>();
  const goalId = Number(params.goalId);
  const goals = useGoals();

  const goal = useMemo(
    () => (goals.data ?? []).find((g) => g.id === goalId),
    [goals.data, goalId]
  );

  const defaultMonths = goal?.targetMonths ?? 12;
  const [targetDate, setTargetDate] = useState<string>(() => addMonths(defaultMonths));
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleConfirm() {
    if (!isValidDate(targetDate)) {
      setError("Enter a valid exam date (YYYY-MM-DD).");
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      await selectGoal(goalId, targetDate);
      router.replace("/(tabs)/goals");
    } catch (err) {
      if (err instanceof FeatureGatedError) {
        setError("This feature isn't available yet. Please check back soon.");
      } else {
        setError(err instanceof Error ? err.message : "Could not select goal");
      }
    } finally {
      setSubmitting(false);
    }
  }

  if (goals.loading && !goals.data) {
    return (
      <View style={styles.center}>
        <ActivityIndicator size="large" />
      </View>
    );
  }

  if (!goal) {
    return (
      <View style={styles.center}>
        <Text style={styles.error}>Goal not found.</Text>
        <TouchableOpacity style={styles.secondaryButton} onPress={() => router.back()}>
          <Text style={styles.secondaryButtonText}>Back</Text>
        </TouchableOpacity>
      </View>
    );
  }

  const presets = [3, 6, 12];

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <TouchableOpacity onPress={() => router.back()}>
        <Text style={styles.back}>← Back</Text>
      </TouchableOpacity>

      <Text style={styles.title}>{goal.name}</Text>
      <Text style={styles.badge}>{goal.examName || "Exam"}</Text>
      <Text style={styles.muted}>{goal.description}</Text>

      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Subjects</Text>
        <View style={styles.chipRow}>
          {goal.subjectAreas.map((s) => (
            <Text key={s} style={styles.metaChip}>
              {s}
            </Text>
          ))}
        </View>
      </View>

      <View style={styles.section}>
        <Text style={styles.sectionTitle}>When is your exam?</Text>
        <View style={styles.chipRow}>
          {presets.map((m) => {
            const value = addMonths(m);
            const active = value === targetDate;
            return (
              <TouchableOpacity
                key={m}
                style={[styles.presetChip, active ? styles.presetActive : null]}
                onPress={() => setTargetDate(value)}
              >
                <Text style={[styles.presetText, active ? styles.presetTextActive : null]}>
                  In {m} months
                </Text>
              </TouchableOpacity>
            );
          })}
        </View>
        <TextInput
          style={styles.input}
          value={targetDate}
          onChangeText={setTargetDate}
          placeholder="YYYY-MM-DD"
          autoCapitalize="none"
          autoCorrect={false}
        />
      </View>

      {error ? <Text style={styles.error}>{error}</Text> : null}

      <TouchableOpacity
        style={[styles.primaryButton, submitting ? styles.disabled : null]}
        onPress={() => void handleConfirm()}
        disabled={submitting}
      >
        <Text style={styles.primaryButtonText}>{submitting ? "Saving…" : "Confirm Goal"}</Text>
      </TouchableOpacity>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: colors.bg },
  content: { padding: 20, gap: 12 },
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: 12, padding: 24, backgroundColor: colors.bg },
  back: { color: colors.brandDark, fontWeight: "700", fontSize: 16 },
  title: { fontSize: 24, fontWeight: "800", color: colors.text },
  muted: { color: colors.textMuted, lineHeight: 20 },
  error: { color: colors.danger },
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
  section: { gap: 8, marginTop: 8 },
  sectionTitle: { fontSize: 16, fontWeight: "800", color: colors.text },
  chipRow: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  metaChip: {
    backgroundColor: colors.surfaceAlt,
    color: colors.textMuted,
    paddingHorizontal: 11,
    paddingVertical: 6,
    borderRadius: radius.sm,
    fontSize: 13,
    fontWeight: "700",
    overflow: "hidden",
  },
  presetChip: {
    borderWidth: 1,
    borderColor: colors.borderStrong,
    borderRadius: 999,
    paddingHorizontal: 14,
    paddingVertical: 8,
    backgroundColor: colors.surface,
  },
  presetActive: { backgroundColor: colors.brand, borderColor: colors.brand },
  presetText: { color: colors.textMuted, fontWeight: "700" },
  presetTextActive: { color: colors.white },
  input: {
    borderWidth: 1,
    borderColor: colors.borderStrong,
    borderRadius: radius.md,
    padding: 12,
    fontSize: 16,
    backgroundColor: colors.surface,
    color: colors.text,
  },
  primaryButton: {
    backgroundColor: colors.brand,
    borderRadius: radius.md,
    paddingVertical: 14,
    alignItems: "center",
    marginTop: 8,
  },
  primaryButtonText: { color: colors.white, fontWeight: "700", fontSize: 16 },
  secondaryButton: {
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.borderStrong,
    borderRadius: radius.md,
    paddingVertical: 12,
    paddingHorizontal: 20,
    alignItems: "center",
  },
  secondaryButtonText: { color: colors.text, fontWeight: "700", fontSize: 16 },
  disabled: { opacity: 0.5 },
});
