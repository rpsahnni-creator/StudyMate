import React, { useCallback, useEffect, useMemo, useState } from "react";
import {
  ActivityIndicator,
  Alert,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from "react-native";
import { router, useLocalSearchParams } from "expo-router";
import { Ionicons } from "@expo/vector-icons";
import {
  getDraft,
  publishDraft,
  saveDraft,
  type DraftDetail,
  type DraftQuestionInput,
} from "../../lib/api";
import { Card, PrimaryButton, SecondaryButton } from "../../components/ui";
import { colors, radius, shadow, spacing } from "../../lib/theme";
import { SkyBackground } from "../../components/SkyBackground";

type QType = "mcq" | "fill_blank" | "true_false";

interface EditableOption {
  text: string;
  isCorrect: boolean;
}

interface EditableQuestion {
  text: string;
  type: QType;
  explanation: string;
  options: EditableOption[];
}

const TYPE_LABELS: Record<QType, string> = {
  mcq: "MCQ",
  fill_blank: "Fill in the blank",
  true_false: "True / False",
};

function labelFor(index: number): string {
  return String.fromCharCode(65 + index);
}

function toEditable(detail: DraftDetail): EditableQuestion[] {
  return detail.questions.map((q) => ({
    text: q.text,
    type: (["mcq", "fill_blank", "true_false"].includes(q.type) ? q.type : "mcq") as QType,
    explanation: q.explanation ?? "",
    options: q.options.map((o) => ({ text: o.text, isCorrect: o.isCorrect })),
  }));
}

function coerceOptionsForType(options: EditableOption[], type: QType): EditableOption[] {
  if (type === "true_false") {
    const correctIdx = options.findIndex((o) => o.isCorrect);
    return [
      { text: "True", isCorrect: correctIdx === 0 },
      { text: "False", isCorrect: correctIdx === 1 },
    ];
  }
  const next = options.filter((o) => o.text.trim() !== "" || true);
  const min = type === "mcq" ? 4 : 2;
  while (next.length < min) next.push({ text: "", isCorrect: false });
  return next;
}

export default function ReviewQuestionsScreen() {
  const params = useLocalSearchParams<{ quizId: string }>();
  const quizId = String(params.quizId);

  const [title, setTitle] = useState("Scanned exam");
  const [questions, setQuestions] = useState<EditableQuestion[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const detail = await getDraft(quizId);
      setTitle(detail.title || "Scanned exam");
      setQuestions(toEditable(detail));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load extracted questions");
    } finally {
      setLoading(false);
    }
  }, [quizId]);

  useEffect(() => {
    void load();
  }, [load]);

  const missingAnswers = useMemo(
    () => questions.filter((q) => !q.options.some((o) => o.isCorrect)).length,
    [questions]
  );

  const update = useCallback((fn: (prev: EditableQuestion[]) => EditableQuestion[]) => {
    setQuestions((prev) => fn(prev.map((q) => ({ ...q, options: q.options.map((o) => ({ ...o })) }))));
  }, []);

  function setQuestionText(qi: number, text: string) {
    update((prev) => {
      prev[qi].text = text;
      return prev;
    });
  }

  function setType(qi: number, type: QType) {
    update((prev) => {
      prev[qi].type = type;
      prev[qi].options = coerceOptionsForType(prev[qi].options, type);
      return prev;
    });
  }

  function setOptionText(qi: number, oi: number, text: string) {
    update((prev) => {
      prev[qi].options[oi].text = text;
      return prev;
    });
  }

  function setCorrect(qi: number, oi: number) {
    update((prev) => {
      prev[qi].options = prev[qi].options.map((o, idx) => ({ ...o, isCorrect: idx === oi }));
      return prev;
    });
  }

  function addOption(qi: number) {
    update((prev) => {
      if (prev[qi].options.length >= 6) return prev;
      prev[qi].options.push({ text: "", isCorrect: false });
      return prev;
    });
  }

  function removeOption(qi: number, oi: number) {
    update((prev) => {
      const min = prev[qi].type === "mcq" ? 2 : 2;
      if (prev[qi].options.length <= min) return prev;
      prev[qi].options.splice(oi, 1);
      return prev;
    });
  }

  function setExplanation(qi: number, text: string) {
    update((prev) => {
      prev[qi].explanation = text;
      return prev;
    });
  }

  function addQuestion() {
    update((prev) => {
      prev.push({
        text: "",
        type: "mcq",
        explanation: "",
        options: [
          { text: "", isCorrect: false },
          { text: "", isCorrect: false },
          { text: "", isCorrect: false },
          { text: "", isCorrect: false },
        ],
      });
      return prev;
    });
  }

  function removeQuestion(qi: number) {
    Alert.alert("Remove question?", "Yeh question exam se hata diya jayega.", [
      { text: "Cancel", style: "cancel" },
      {
        text: "Remove",
        style: "destructive",
        onPress: () => update((prev) => prev.filter((_, idx) => idx !== qi)),
      },
    ]);
  }

  function buildPayload(): DraftQuestionInput[] {
    return questions.map((q) => ({
      text: q.text.trim(),
      type: q.type,
      explanation: q.explanation.trim(),
      correctIndex: q.options.findIndex((o) => o.isCorrect),
      options: q.options
        .map((o, idx) => ({ label: labelFor(idx), text: o.text.trim() }))
        .filter((o) => o.text !== ""),
    }));
  }

  function validate(): string | null {
    if (questions.length === 0) return "Kam se kam ek question chahiye.";
    for (let i = 0; i < questions.length; i++) {
      const q = questions[i];
      if (q.text.trim() === "") return `Question ${i + 1} ka text khali hai.`;
      const filled = q.options.filter((o) => o.text.trim() !== "");
      if (filled.length < 2) return `Question ${i + 1} me kam se kam 2 options chahiye.`;
    }
    return null;
  }

  const doSave = useCallback(
    async (silent = false): Promise<boolean> => {
      const v = validate();
      if (v) {
        if (!silent) Alert.alert("Check karein", v);
        return false;
      }
      setBusy(true);
      setError(null);
      try {
        const detail = await saveDraft(quizId, { questions: buildPayload() });
        setQuestions(toEditable(detail));
        if (!silent) Alert.alert("Saved", "Aapke changes save ho gaye.");
        return true;
      } catch (err) {
        const message = err instanceof Error ? err.message : "Save failed";
        setError(message);
        if (!silent) Alert.alert("Save failed", message);
        return false;
      } finally {
        setBusy(false);
      }
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [questions, quizId]
  );

  async function doPublish() {
    if (missingAnswers > 0) {
      Alert.alert(
        "Answers baaki hain",
        `${missingAnswers} question(s) ka sahi jawab select karein, tabhi exam publish hoga.`
      );
      return;
    }
    const saved = await doSave(true);
    if (!saved) return;
    setBusy(true);
    try {
      await publishDraft(quizId);
      Alert.alert("Exam published!", "Ab students yeh exam de sakte hain.", [
        { text: "OK", onPress: () => router.replace(`/quiz/${quizId}`) },
      ]);
    } catch (err) {
      const message = err instanceof Error ? err.message : "Publish failed";
      setError(message);
      Alert.alert("Publish failed", message);
    } finally {
      setBusy(false);
    }
  }

  if (loading) {
    return (
      <SkyBackground>
        <View style={styles.center}>
          <ActivityIndicator size="large" color={colors.brand} />
          <Text style={styles.muted}>Loading extracted questions…</Text>
        </View>
      </SkyBackground>
    );
  }

  if (error && questions.length === 0) {
    return (
      <SkyBackground>
        <View style={styles.center}>
          <Text style={styles.error}>{error}</Text>
          <PrimaryButton title="Retry" icon="refresh-outline" onPress={() => void load()} />
        </View>
      </SkyBackground>
    );
  }

  return (
    <SkyBackground>
      <ScrollView style={styles.screen} contentContainerStyle={styles.container} keyboardShouldPersistTaps="handled">
        <Card glass style={styles.headerCard}>
          <Text style={styles.title}>{title}</Text>
          <Text style={styles.sub}>
            Scan se {questions.length} question mile. Review karein, zaroorat ho to edit karein, aur har question ka sahi
            jawab select karein.
          </Text>
          <View style={[styles.answerPill, missingAnswers > 0 ? styles.answerPillWarn : styles.answerPillOk]}>
            <Ionicons
              name={missingAnswers > 0 ? "alert-circle-outline" : "checkmark-circle-outline"}
              size={16}
              color={missingAnswers > 0 ? colors.warning : colors.success}
            />
            <Text style={[styles.answerPillText, { color: missingAnswers > 0 ? colors.warning : colors.success }]}>
              {missingAnswers > 0 ? `${missingAnswers} answer baaki` : "Sabhi answers set hain"}
            </Text>
          </View>
        </Card>

        {questions.map((q, qi) => {
          const noAnswer = !q.options.some((o) => o.isCorrect);
          return (
            <Card glass key={qi} style={styles.qCard}>
              <View style={styles.qHead}>
                <Text style={styles.qNum}>Q{qi + 1}</Text>
                <Pressable style={styles.deleteBtn} onPress={() => removeQuestion(qi)} hitSlop={8}>
                  <Ionicons name="trash-outline" size={18} color={colors.danger} />
                </Pressable>
              </View>

              <View style={styles.typeRow}>
                {(Object.keys(TYPE_LABELS) as QType[]).map((t) => (
                  <Pressable
                    key={t}
                    onPress={() => setType(qi, t)}
                    style={[styles.typeChip, q.type === t ? styles.typeChipActive : null]}
                  >
                    <Text style={[styles.typeChipText, q.type === t ? styles.typeChipTextActive : null]}>
                      {TYPE_LABELS[t]}
                    </Text>
                  </Pressable>
                ))}
              </View>

              <TextInput
                style={styles.input}
                value={q.text}
                onChangeText={(t) => setQuestionText(qi, t)}
                placeholder="Question text"
                placeholderTextColor={colors.textSubtle}
                multiline
              />

              <Text style={styles.optionsLabel}>Options — sahi jawab par tap karein</Text>
              {q.options.map((o, oi) => (
                <View key={oi} style={styles.optionRow}>
                  <Pressable
                    onPress={() => setCorrect(qi, oi)}
                    style={[styles.radio, o.isCorrect ? styles.radioOn : null]}
                    hitSlop={6}
                  >
                    {o.isCorrect ? <Ionicons name="checkmark" size={14} color={colors.white} /> : null}
                  </Pressable>
                  <Text style={styles.optLabel}>{labelFor(oi)}</Text>
                  <TextInput
                    style={[styles.input, styles.optInput]}
                    value={o.text}
                    onChangeText={(t) => setOptionText(qi, oi, t)}
                    placeholder={`Option ${labelFor(oi)}`}
                    placeholderTextColor={colors.textSubtle}
                    editable={q.type !== "true_false"}
                  />
                  {q.type !== "true_false" && q.options.length > 2 ? (
                    <Pressable onPress={() => removeOption(qi, oi)} hitSlop={6}>
                      <Ionicons name="close-circle" size={20} color={colors.textSubtle} />
                    </Pressable>
                  ) : null}
                </View>
              ))}

              {q.type !== "true_false" && q.options.length < 6 ? (
                <Pressable style={styles.addOptBtn} onPress={() => addOption(qi)}>
                  <Ionicons name="add" size={16} color={colors.brandDark} />
                  <Text style={styles.addOptText}>Add option</Text>
                </Pressable>
              ) : null}

              {noAnswer ? <Text style={styles.warnText}>Is question ka sahi jawab select karein.</Text> : null}

              <TextInput
                style={[styles.input, styles.explInput]}
                value={q.explanation}
                onChangeText={(t) => setExplanation(qi, t)}
                placeholder="Explanation (optional — publish par AI khud bhi bana dega)"
                placeholderTextColor={colors.textSubtle}
                multiline
              />
            </Card>
          );
        })}

        <SecondaryButton title="Add question" icon="add-circle-outline" onPress={addQuestion} />

        {error ? <Text style={styles.error}>{error}</Text> : null}

        <View style={styles.footer}>
          <SecondaryButton
            title="Save draft"
            icon="save-outline"
            onPress={() => void doSave(false)}
            style={styles.flexBtn}
          />
          <PrimaryButton
            title={busy ? "Working…" : "Publish exam"}
            icon="checkmark-done-outline"
            onPress={() => void doPublish()}
            disabled={busy}
            loading={busy}
            style={styles.flexBtn}
          />
        </View>
      </ScrollView>
    </SkyBackground>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1 },
  container: { padding: spacing.lg, gap: spacing.md, paddingBottom: 48 },
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: 12, padding: 24 },
  muted: { color: colors.textMuted, fontSize: 15 },
  error: { color: colors.danger, fontSize: 14 },
  headerCard: { gap: 8 },
  title: { fontSize: 20, fontWeight: "800", color: colors.text },
  sub: { fontSize: 13.5, color: colors.textMuted, lineHeight: 19 },
  answerPill: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
    alignSelf: "flex-start",
    paddingHorizontal: 10,
    paddingVertical: 5,
    borderRadius: radius.full,
    marginTop: 2,
  },
  answerPillWarn: { backgroundColor: colors.warningBg },
  answerPillOk: { backgroundColor: colors.successBg },
  answerPillText: { fontSize: 12.5, fontWeight: "700" },
  qCard: { gap: 10 },
  qHead: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  qNum: { fontSize: 15, fontWeight: "800", color: colors.brandDark },
  deleteBtn: { padding: 4 },
  typeRow: { flexDirection: "row", gap: 6, flexWrap: "wrap" },
  typeChip: {
    paddingHorizontal: 10,
    paddingVertical: 6,
    borderRadius: radius.full,
    backgroundColor: colors.surfaceAlt,
    borderWidth: 1,
    borderColor: colors.border,
  },
  typeChipActive: { backgroundColor: colors.brandDark, borderColor: colors.brandDark },
  typeChipText: { fontSize: 12, fontWeight: "700", color: colors.text },
  typeChipTextActive: { color: colors.white },
  input: {
    borderWidth: 1,
    borderColor: colors.borderStrong,
    borderRadius: radius.md,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 15,
    color: colors.text,
    backgroundColor: colors.surface,
  },
  optionsLabel: { fontSize: 12.5, fontWeight: "700", color: colors.textMuted, marginTop: 2 },
  optionRow: { flexDirection: "row", alignItems: "center", gap: 8 },
  optInput: { flex: 1, paddingVertical: 8 },
  optLabel: { width: 16, fontWeight: "800", color: colors.textMuted },
  radio: {
    width: 26,
    height: 26,
    borderRadius: 13,
    borderWidth: 2,
    borderColor: colors.borderStrong,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.surface,
  },
  radioOn: { backgroundColor: colors.success, borderColor: colors.success },
  addOptBtn: { flexDirection: "row", alignItems: "center", gap: 4, alignSelf: "flex-start", paddingVertical: 4 },
  addOptText: { color: colors.brandDark, fontWeight: "700", fontSize: 13 },
  warnText: { color: colors.warning, fontSize: 12.5, fontWeight: "600" },
  explInput: { minHeight: 44, marginTop: 2 },
  footer: { flexDirection: "row", gap: spacing.md, marginTop: spacing.sm },
  flexBtn: { flex: 1 },
});
