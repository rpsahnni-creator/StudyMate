import React, { useMemo, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  ScrollView,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import { router } from "expo-router";
import { useTodayPractice } from "../../hooks/useCareerGoals";
import {
  submitPractice,
  type PracticeAnswerInput,
  type SubmitPracticeResult,
  type TodayPractice,
} from "../../lib/careerGoals";
import { colors, radius, shadow, scoreColor } from "../../lib/theme";

export default function PracticeScreen() {
  const practice = useTodayPractice();
  const [answers, setAnswers] = useState<Record<number, number | null>>({});
  const [current, setCurrent] = useState(0);
  const [submitting, setSubmitting] = useState(false);
  const [result, setResult] = useState<SubmitPracticeResult | null>(null);

  if (practice.gated) {
    return (
      <View style={styles.center}>
        <Text style={styles.title}>Coming Soon</Text>
        <Text style={styles.muted}>Daily practice isn't available yet.</Text>
      </View>
    );
  }

  if (practice.loading && !practice.data) {
    return (
      <View style={styles.center}>
        <ActivityIndicator size="large" />
        <Text style={styles.muted}>Preparing today's practice…</Text>
      </View>
    );
  }

  if (practice.error) {
    return (
      <View style={styles.center}>
        <Text style={styles.error}>{practice.error}</Text>
        <TouchableOpacity style={styles.primaryButton} onPress={practice.reload}>
          <Text style={styles.primaryButtonText}>Retry</Text>
        </TouchableOpacity>
      </View>
    );
  }

  const data = practice.data;
  if (!data) return null;

  if (result) {
    return <ResultView result={result} />;
  }

  if (data.status === "completed") {
    return <AlreadyDoneView data={data} />;
  }

  return (
    <PracticeRunner
      data={data}
      answers={answers}
      setAnswers={setAnswers}
      current={current}
      setCurrent={setCurrent}
      submitting={submitting}
      onSubmit={async () => {
        setSubmitting(true);
        try {
          const payload: PracticeAnswerInput[] = data.questions.map((q) => ({
            questionId: q.id,
            selectedOptionId: answers[q.id] ?? null,
          }));
          const res = await submitPractice(data.setId, payload);
          setResult(res);
        } catch (err) {
          Alert.alert("Submit failed", err instanceof Error ? err.message : "Please try again.");
        } finally {
          setSubmitting(false);
        }
      }}
    />
  );
}

function PracticeRunner({
  data,
  answers,
  setAnswers,
  current,
  setCurrent,
  submitting,
  onSubmit,
}: {
  data: TodayPractice;
  answers: Record<number, number | null>;
  setAnswers: React.Dispatch<React.SetStateAction<Record<number, number | null>>>;
  current: number;
  setCurrent: React.Dispatch<React.SetStateAction<number>>;
  submitting: boolean;
  onSubmit: () => void | Promise<void>;
}) {
  const total = data.questions.length;

  if (total === 0) {
    return (
      <View style={styles.center}>
        <Text style={styles.muted}>No questions available for today's practice yet.</Text>
        <TouchableOpacity style={styles.secondaryButton} onPress={() => router.back()}>
          <Text style={styles.secondaryButtonText}>Back</Text>
        </TouchableOpacity>
      </View>
    );
  }

  const question = data.questions[current];
  const progress = ((current + 1) / total) * 100;

  function confirmSubmit() {
    const answered = Object.values(answers).filter((v) => v != null).length;
    Alert.alert("Submit practice?", `You answered ${answered} of ${total} questions.`, [
      { text: "Cancel", style: "cancel" },
      { text: "Submit", onPress: () => void onSubmit() },
    ]);
  }

  return (
    <View style={styles.container}>
      <View style={styles.header}>
        <Text style={styles.progressText}>
          Q {current + 1} / {total}
        </Text>
        <Text style={styles.topicBadge}>{question.topic || question.subject || "Practice"}</Text>
      </View>

      {data.topicFocus.length > 0 ? (
        <Text style={styles.focus}>Focus: {data.topicFocus.join(", ")}</Text>
      ) : null}

      <View style={styles.progressTrack}>
        <View style={[styles.progressFill, { width: `${progress}%` }]} />
      </View>

      <ScrollView style={styles.body} contentContainerStyle={styles.bodyContent}>
        <Text style={styles.questionText}>{question.text}</Text>

        {question.options.map((opt, idx) => {
          const selected = answers[question.id] === opt.id;
          return (
            <TouchableOpacity
              key={opt.id}
              style={[styles.option, selected ? styles.optionSelected : null]}
              onPress={() => setAnswers((prev) => ({ ...prev, [question.id]: opt.id }))}
            >
              <Text style={styles.optionLabel}>{opt.label || String.fromCharCode(65 + idx)}</Text>
              <Text style={styles.optionText}>{opt.text}</Text>
            </TouchableOpacity>
          );
        })}
      </ScrollView>

      <View style={styles.navRow}>
        <TouchableOpacity
          style={[styles.secondaryButton, current === 0 ? styles.disabled : null]}
          onPress={() => setCurrent((c) => Math.max(0, c - 1))}
          disabled={current === 0}
        >
          <Text style={styles.secondaryButtonText}>Previous</Text>
        </TouchableOpacity>

        {current < total - 1 ? (
          <TouchableOpacity
            style={styles.secondaryButton}
            onPress={() => setCurrent((c) => Math.min(total - 1, c + 1))}
          >
            <Text style={styles.secondaryButtonText}>Next</Text>
          </TouchableOpacity>
        ) : (
          <TouchableOpacity
            style={[styles.primaryButton, submitting ? styles.disabled : null]}
            onPress={confirmSubmit}
            disabled={submitting}
          >
            <Text style={styles.primaryButtonText}>{submitting ? "Submitting…" : "Submit"}</Text>
          </TouchableOpacity>
        )}
      </View>
    </View>
  );
}

function ResultView({ result }: { result: SubmitPracticeResult }) {
  const improved = useMemo(
    () => result.skillUpdates.filter((s) => s.direction === "improved"),
    [result.skillUpdates]
  );
  const needsWork = useMemo(
    () => result.skillUpdates.filter((s) => s.direction === "needs_work"),
    [result.skillUpdates]
  );

  const sc = scoreColor(result.score);

  return (
    <ScrollView style={styles.screen} contentContainerStyle={styles.resultContent}>
      <Text style={styles.emoji}>{result.score >= 60 ? "🎉" : "💪"}</Text>
      <Text style={styles.title}>Practice Complete</Text>
      <View style={[styles.scoreCircle, { borderColor: sc, backgroundColor: `${sc}14` }]}>
        <Text style={[styles.scoreText, { color: sc }]}>{result.score.toFixed(0)}%</Text>
      </View>
      <Text style={styles.muted}>
        {result.correctCount} correct · {result.wrongCount} wrong · {result.skippedCount} skipped
      </Text>

      {improved.map((s) => (
        <Text key={`up-${s.topic}`} style={styles.improved}>
          You improved in {s.topic}! 🎉
        </Text>
      ))}
      {needsWork.map((s) => (
        <Text key={`down-${s.topic}`} style={styles.needsWork}>
          Keep working on {s.topic}.
        </Text>
      ))}

      <View style={styles.resultButtons}>
        <TouchableOpacity style={styles.primaryButton} onPress={() => router.replace("/goals/skills")}>
          <Text style={styles.primaryButtonText}>View Skills</Text>
        </TouchableOpacity>
        <TouchableOpacity style={styles.secondaryButton} onPress={() => router.replace("/(tabs)/goals")}>
          <Text style={styles.secondaryButtonText}>Done</Text>
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
}

function AlreadyDoneView({ data }: { data: TodayPractice }) {
  const sc = data.score != null ? scoreColor(data.score) : colors.brand;
  return (
    <View style={styles.center}>
      <Text style={styles.title}>Already Done Today</Text>
      <View style={[styles.scoreCircle, { borderColor: sc, backgroundColor: `${sc}14` }]}>
        <Text style={[styles.scoreText, { color: sc }]}>{data.score != null ? `${data.score.toFixed(0)}%` : "✓"}</Text>
      </View>
      <Text style={styles.muted}>Come back tomorrow for a new set.</Text>
      <View style={styles.resultButtons}>
        <TouchableOpacity style={styles.primaryButton} onPress={() => router.replace("/goals/skills")}>
          <Text style={styles.primaryButtonText}>View Skills</Text>
        </TouchableOpacity>
        <TouchableOpacity style={styles.secondaryButton} onPress={() => router.replace("/(tabs)/goals")}>
          <Text style={styles.secondaryButtonText}>Back</Text>
        </TouchableOpacity>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: colors.bg },
  container: { flex: 1, padding: 20, gap: 12, backgroundColor: colors.bg },
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: 12, padding: 24, backgroundColor: colors.bg },
  emoji: { fontSize: 44 },
  title: { fontSize: 23, fontWeight: "800", color: colors.text },
  muted: { color: colors.textMuted, textAlign: "center", lineHeight: 20 },
  error: { color: colors.danger },
  header: { flexDirection: "row", justifyContent: "space-between", alignItems: "center" },
  progressText: { fontWeight: "700", color: colors.textMuted },
  topicBadge: {
    backgroundColor: colors.brandSoft,
    color: colors.brandDark,
    paddingHorizontal: 11,
    paddingVertical: 4,
    borderRadius: 999,
    fontSize: 12,
    fontWeight: "700",
    overflow: "hidden",
  },
  focus: { color: colors.textMuted, fontStyle: "italic" },
  progressTrack: { height: 8, borderRadius: radius.full, backgroundColor: colors.surfaceAlt, overflow: "hidden" },
  progressFill: { height: "100%", backgroundColor: colors.brand, borderRadius: radius.full },
  body: { flex: 1 },
  bodyContent: { gap: 12, paddingVertical: 12 },
  questionText: { fontSize: 19, fontWeight: "700", color: colors.text, marginBottom: 8, lineHeight: 26 },
  option: {
    flexDirection: "row",
    alignItems: "center",
    gap: 12,
    borderWidth: 2,
    borderColor: colors.borderStrong,
    borderRadius: radius.md,
    padding: 14,
    backgroundColor: colors.surface,
  },
  optionSelected: { borderColor: colors.brand, backgroundColor: colors.brandSoft },
  optionLabel: {
    width: 30,
    height: 30,
    borderRadius: 999,
    backgroundColor: colors.brandSoft,
    color: colors.brandDark,
    textAlign: "center",
    lineHeight: 30,
    fontWeight: "800",
    overflow: "hidden",
  },
  optionText: { flex: 1, fontSize: 15, color: colors.text },
  navRow: { flexDirection: "row", justifyContent: "space-between", gap: 12 },
  primaryButton: {
    backgroundColor: colors.brand,
    borderRadius: radius.md,
    paddingVertical: 13,
    paddingHorizontal: 20,
    alignItems: "center",
    flex: 1,
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
    flex: 1,
  },
  secondaryButtonText: { color: colors.text, fontWeight: "700", fontSize: 16 },
  disabled: { opacity: 0.5 },
  resultContent: { padding: 24, alignItems: "center", gap: 14 },
  scoreCircle: {
    width: 156,
    height: 156,
    borderRadius: 999,
    borderWidth: 7,
    alignItems: "center",
    justifyContent: "center",
  },
  scoreText: { fontSize: 36, fontWeight: "800" },
  improved: { color: colors.success, fontWeight: "800", fontSize: 16 },
  needsWork: { color: "#b45309", fontWeight: "700", fontSize: 15 },
  resultButtons: { flexDirection: "row", gap: 12, marginTop: 12, alignSelf: "stretch" },
});
