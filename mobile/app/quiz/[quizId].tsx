import React, { useCallback, useEffect, useRef, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  ScrollView,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import { router, useLocalSearchParams } from "expo-router";
import AsyncStorage from "@react-native-async-storage/async-storage";
import {
  getQuiz,
  startAttempt,
  submitAttempt,
  type AnswerInput,
  type QuizDetail,
} from "../../lib/api";
import { getAccessToken } from "../../lib/auth";
import { colors, radius, shadow } from "../../lib/theme";
import { SkyBackground } from "../../components/SkyBackground";

function answersKey(attemptId: string): string {
  return `studyapp.quiz.answers.${attemptId}`;
}

function questionTypeLabel(type?: string): string {
  switch (type) {
    case "fill_blank":
      return "Fill in the blank";
    case "true_false":
      return "True / False";
    default:
      return "MCQ";
  }
}

export default function QuizPlayScreen() {
  const params = useLocalSearchParams<{ quizId: string }>();
  const quizId = String(params.quizId);

  const [quiz, setQuiz] = useState<QuizDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [attemptId, setAttemptId] = useState<string | null>(null);
  const [starting, setStarting] = useState(false);
  const [current, setCurrent] = useState(0);
  const [answers, setAnswers] = useState<Record<number, number | null>>({});
  const [timeLeft, setTimeLeft] = useState<number | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);

  const submitLockRef = useRef(false);

  const loadQuiz = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getQuiz(quizId);
      setQuiz(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load quiz");
    } finally {
      setLoading(false);
    }
  }, [quizId]);

  useEffect(() => {
    void loadQuiz();
    void getAccessToken();
  }, [loadQuiz]);

  useEffect(() => {
    if (!quiz?.questions?.length) return;
    const first = quiz.questions[0]?.text ?? "";
    if (first.includes("Stub question") || first.includes("Distractor option")) {
      Alert.alert(
        "Purana quiz",
        "Yeh purana cached quiz hai. Sahi questions ke liye Scan tab par jao aur naya page scan karo.",
        [{ text: "Scan karo", onPress: () => router.replace("/(tabs)/scan") }]
      );
    }
  }, [quiz]);

  async function handleStart() {
    if (!quiz) return;
    setStarting(true);
    setError(null);
    try {
      const { attemptId: newAttemptId } = await startAttempt(quizId);
      const id = String(newAttemptId);
      setAttemptId(id);
      setTimeLeft(quiz.timeLimit);
      const saved = await AsyncStorage.getItem(answersKey(id));
      if (saved) {
        setAnswers(JSON.parse(saved) as Record<number, number | null>);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not start quiz");
    } finally {
      setStarting(false);
    }
  }

  const buildAnswers = useCallback((): AnswerInput[] => {
    if (!quiz) return [];
    return quiz.questions.map((q) => ({
      questionId: q.id,
      selectedOptionId: answers[q.id] ?? null,
    }));
  }, [quiz, answers]);

  const doSubmit = useCallback(async () => {
    if (submitLockRef.current || submitted || !attemptId) return;
    submitLockRef.current = true;
    setSubmitting(true);
    setError(null);

    try {
      const result = await submitAttempt(quizId, attemptId, buildAnswers());
      setSubmitted(true);
      router.replace({
        pathname: "/quiz/result",
        params: {
          quizId,
          attemptId,
          score: String(result.score),
          correct: String(result.correctCount),
          wrong: String(result.wrongCount),
          skipped: String(result.skippedCount),
          total: String(result.totalQuestions),
          timeTaken: String(result.timeTaken),
        },
      });
      void AsyncStorage.removeItem(answersKey(attemptId));
    } catch (err) {
      submitLockRef.current = false;
      setSubmitting(false);
      const message = err instanceof Error ? err.message : "Submit failed";
      if (message.toLowerCase().includes("network") || message.toLowerCase().includes("connect")) {
        Alert.alert("No connection", "Your answers are saved on this device. Reconnect to submit.");
      } else {
        setError(message);
      }
    }
  }, [quizId, attemptId, buildAnswers, submitted]);

  // Countdown timer with auto-submit at zero.
  useEffect(() => {
    if (timeLeft === null || submitted || !attemptId) return;
    if (timeLeft <= 0) {
      void doSubmit();
      return;
    }
    const id = setTimeout(() => setTimeLeft((t) => (t === null ? t : t - 1)), 1000);
    return () => clearTimeout(id);
  }, [timeLeft, submitted, attemptId, doSubmit]);

  const selectOption = useCallback(
    (questionId: number, optionId: number) => {
      if (submitted || !attemptId) return;
      setAnswers((prev) => {
        const next = { ...prev, [questionId]: optionId };
        void AsyncStorage.setItem(answersKey(attemptId), JSON.stringify(next));
        return next;
      });
    },
    [attemptId, submitted]
  );

  function confirmSubmit() {
    const answered = Object.values(answers).filter((v) => v != null).length;
    Alert.alert(
      "Submit quiz?",
      `You answered ${answered} of ${quiz?.questions.length ?? 0} questions.`,
      [
        { text: "Cancel", style: "cancel" },
        { text: "Submit", onPress: () => void doSubmit() },
      ]
    );
  }

  if (loading) {
    return (
      <SkyBackground>
        <View style={styles.center}>
          <ActivityIndicator size="large" />
          <Text style={styles.muted}>Loading quiz…</Text>
        </View>
      </SkyBackground>
    );
  }

  if (error && !quiz) {
    return (
      <SkyBackground>
        <View style={styles.center}>
          <Text style={styles.error}>{error}</Text>
          <TouchableOpacity style={styles.primaryButton} onPress={() => void loadQuiz()}>
            <Text style={styles.primaryButtonText}>Retry</Text>
          </TouchableOpacity>
        </View>
      </SkyBackground>
    );
  }

  if (!quiz) return null;

  // Pre-start screen.
  if (!attemptId) {
    return (
      <SkyBackground>
        <View style={styles.container}>
          <Text style={styles.title}>{quiz.title}</Text>
          <View style={styles.badgeRow}>
            <Text style={styles.badge}>{quiz.subject || "General"}</Text>
            <Text style={styles.badge}>{quiz.board || "—"}</Text>
          </View>
          <Text style={styles.muted}>{quiz.totalQuestions} questions</Text>
          <Text style={styles.muted}>Time limit: {Math.round(quiz.timeLimit / 60)} min</Text>
          {error ? <Text style={styles.error}>{error}</Text> : null}
          <TouchableOpacity style={styles.primaryButton} onPress={() => void handleStart()} disabled={starting}>
            <Text style={styles.primaryButtonText}>{starting ? "Starting…" : "Start Quiz"}</Text>
          </TouchableOpacity>
        </View>
      </SkyBackground>
    );
  }

  const question = quiz.questions[current];
  const minutes = Math.floor((timeLeft ?? 0) / 60);
  const seconds = (timeLeft ?? 0) % 60;
  const progress = ((current + 1) / quiz.questions.length) * 100;

  return (
    <SkyBackground>
      <View style={styles.container}>
        <View style={styles.header}>
          <Text style={styles.progressText}>
            Q {current + 1} / {quiz.questions.length}
          </Text>
          <Text style={[styles.timer, (timeLeft ?? 0) <= 30 ? styles.timerDanger : null]}>
            {minutes}:{seconds.toString().padStart(2, "0")}
          </Text>
        </View>

        <View style={styles.progressTrack}>
          <View style={[styles.progressFill, { width: `${progress}%` }]} />
        </View>

        <ScrollView style={styles.body} contentContainerStyle={styles.bodyContent}>
          <Text style={styles.typeBadge}>{questionTypeLabel(question.type)}</Text>
          <Text style={styles.questionText}>{question.text}</Text>

          {question.options.map((opt, idx) => {
            const selected = answers[question.id] === opt.id;
            return (
              <TouchableOpacity
                key={opt.id}
                style={[styles.option, selected ? styles.optionSelected : null]}
                onPress={() => selectOption(question.id, opt.id)}
                disabled={submitted}
              >
                <Text style={styles.optionLabel}>{opt.label || String.fromCharCode(65 + idx)}</Text>
                <Text style={styles.optionText}>{opt.text}</Text>
              </TouchableOpacity>
            );
          })}

          {error ? <Text style={styles.error}>{error}</Text> : null}
        </ScrollView>

        <View style={styles.footer}>
          <TouchableOpacity
            style={[styles.footerBtn, styles.footerBtnSecondary, current === 0 ? styles.disabled : null]}
            onPress={() => setCurrent((c) => Math.max(0, c - 1))}
            disabled={current === 0 || submitted}
          >
            <Text style={styles.footerBtnSecondaryText}>Prev</Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={[
              styles.footerBtn,
              styles.footerBtnSecondary,
              current >= quiz.questions.length - 1 ? styles.disabled : null,
            ]}
            onPress={() => setCurrent((c) => Math.min(quiz.questions.length - 1, c + 1))}
            disabled={current >= quiz.questions.length - 1 || submitted}
          >
            <Text style={styles.footerBtnSecondaryText}>Next</Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={[styles.footerBtn, styles.footerBtnSubmit, submitting || submitted ? styles.disabled : null]}
            onPress={confirmSubmit}
            disabled={submitting || submitted}
          >
            <Text style={styles.footerBtnSubmitText}>{submitting ? "…" : "Submit"}</Text>
          </TouchableOpacity>
        </View>
      </View>
    </SkyBackground>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, padding: 20, gap: 14, backgroundColor: "transparent" },
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: 12, padding: 24, backgroundColor: "transparent" },
  title: { fontSize: 24, fontWeight: "800", color: colors.text },
  badgeRow: { flexDirection: "row", gap: 8 },
  badge: {
    backgroundColor: colors.brandSoft,
    color: colors.brandDark,
    paddingHorizontal: 12,
    paddingVertical: 5,
    borderRadius: 999,
    fontSize: 13,
    fontWeight: "700",
    overflow: "hidden",
  },
  muted: { color: colors.textMuted, fontSize: 15 },
  header: { flexDirection: "row", justifyContent: "space-between", alignItems: "center" },
  progressText: { fontWeight: "700", color: colors.textMuted },
  timer: {
    fontSize: 18,
    fontWeight: "800",
    color: colors.text,
    backgroundColor: colors.surface,
    paddingHorizontal: 12,
    paddingVertical: 3,
    borderRadius: 999,
    overflow: "hidden",
    ...shadow.sm,
  },
  timerDanger: { color: colors.danger },
  footer: {
    flexDirection: "row",
    gap: 10,
    paddingTop: 4,
    paddingBottom: 8,
  },
  footerBtn: {
    flex: 1,
    borderRadius: radius.md,
    paddingVertical: 14,
    alignItems: "center",
    justifyContent: "center",
    minHeight: 48,
  },
  footerBtnSecondary: {
    backgroundColor: colors.surface,
    borderWidth: 1,
    borderColor: colors.borderStrong,
  },
  footerBtnSecondaryText: { color: colors.text, fontWeight: "700", fontSize: 15 },
  footerBtnSubmit: {
    backgroundColor: colors.brand,
    flex: 1.2,
    ...shadow.brand,
  },
  footerBtnSubmitText: { color: colors.white, fontWeight: "800", fontSize: 15 },
  progressTrack: { height: 8, borderRadius: radius.full, backgroundColor: colors.surfaceAlt, overflow: "hidden" },
  progressFill: { height: "100%", backgroundColor: colors.brand, borderRadius: radius.full },
  body: { flex: 1 },
  bodyContent: { gap: 12, paddingVertical: 12 },
  questionText: { fontSize: 19, fontWeight: "700", color: colors.text, marginBottom: 8, lineHeight: 26 },
  typeBadge: {
    alignSelf: "flex-start",
    fontSize: 11,
    fontWeight: "800",
    color: colors.brandDark,
    backgroundColor: colors.brandSoft,
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: radius.full,
    marginBottom: 10,
    overflow: "hidden",
  },
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
  error: { color: colors.danger },
});
