import React, { useCallback, useEffect, useState } from "react";
import {
  ActivityIndicator,
  FlatList,
  StyleSheet,
  Text,
  TouchableOpacity,
  View,
} from "react-native";
import { useLocalSearchParams } from "expo-router";
import { getReview, type ReviewDetail, type ReviewQuestion } from "../../lib/api";
import { colors, radius, shadow } from "../../lib/theme";

export default function QuizReviewScreen() {
  const params = useLocalSearchParams<{ quizId: string; attemptId: string }>();
  const quizId = String(params.quizId);
  const attemptId = String(params.attemptId);

  const [review, setReview] = useState<ReviewDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<Record<number, boolean>>({});

  const loadReview = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await getReview(quizId, attemptId);
      setReview(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load review");
    } finally {
      setLoading(false);
    }
  }, [quizId, attemptId]);

  useEffect(() => {
    void loadReview();
  }, [loadReview]);

  if (loading) {
    return (
      <View style={styles.center}>
        <ActivityIndicator size="large" />
        <Text style={styles.muted}>Loading review…</Text>
      </View>
    );
  }

  if (error || !review) {
    return (
      <View style={styles.center}>
        <Text style={styles.error}>{error ?? "No review found"}</Text>
        <TouchableOpacity style={styles.retryButton} onPress={() => void loadReview()}>
          <Text style={styles.retryText}>Retry</Text>
        </TouchableOpacity>
      </View>
    );
  }

  const correctCount = review.questions.filter((q) => q.status === "correct").length;

  return (
    <FlatList
      style={styles.list}
      contentContainerStyle={styles.listContent}
      data={review.questions}
      keyExtractor={(q: ReviewQuestion) => String(q.id)}
      ListHeaderComponent={
        <View style={styles.summary}>
          <Text style={styles.title}>Review</Text>
          <Text style={styles.summaryText}>
            Score {review.score.toFixed(1)}% · {correctCount} / {review.questions.length} correct
          </Text>
        </View>
      }
      renderItem={({ item, index }: { item: ReviewQuestion; index: number }) => (
        <ReviewCard
          question={item}
          index={index}
          expanded={!!expanded[item.id]}
          onToggle={() => setExpanded((e) => ({ ...e, [item.id]: !e[item.id] }))}
        />
      )}
    />
  );
}

function ReviewCard({
  question,
  index,
  expanded,
  onToggle,
}: {
  question: ReviewQuestion;
  index: number;
  expanded: boolean;
  onToggle: () => void;
}) {
  const statusColor =
    question.status === "correct" ? colors.success : question.status === "wrong" ? colors.danger : colors.warning;

  return (
    <View style={[styles.card, { borderLeftWidth: 4, borderLeftColor: statusColor }]}>
      <View style={styles.cardHeader}>
        <Text style={styles.qNumber}>Q{index + 1}</Text>
        <Text style={[styles.statusPill, { backgroundColor: statusColor }]}>{question.status}</Text>
      </View>
      <Text style={styles.qText}>{question.text}</Text>

      <View style={styles.answerRow}>
        <Text style={styles.answerLabel}>Your answer: </Text>
        <Text
          style={[
            styles.answerValue,
            {
              color:
                question.status === "correct"
                  ? "#16a34a"
                  : question.status === "wrong"
                  ? "#dc2626"
                  : "#666",
            },
          ]}
        >
          {question.yourAnswerText || (question.yourAnswer != null ? `Option #${question.yourAnswer}` : "Skipped")}
        </Text>
      </View>

      {!question.isCorrect && question.correctAnswer != null ? (
        <View style={styles.answerRow}>
          <Text style={styles.answerLabel}>Correct answer: </Text>
          <Text style={[styles.answerValue, { color: "#16a34a" }]}>
            {question.correctAnswerText || (question.correctAnswer != null ? `Option #${question.correctAnswer}` : "—")}
          </Text>
        </View>
      ) : null}

      {question.explanation ? (
        <View style={styles.explanationWrap}>
          <TouchableOpacity onPress={onToggle}>
            <Text style={styles.explanationToggle}>
              {expanded ? "Hide explanation" : "Show explanation"}
            </Text>
          </TouchableOpacity>
          {expanded ? <Text style={styles.explanationText}>{question.explanation}</Text> : null}
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  list: { flex: 1, backgroundColor: colors.bg },
  listContent: { padding: 20, gap: 14 },
  center: { flex: 1, alignItems: "center", justifyContent: "center", gap: 12, padding: 24, backgroundColor: colors.bg },
  muted: { color: colors.textMuted },
  summary: { marginBottom: 6 },
  title: { fontSize: 26, fontWeight: "800", color: colors.text, marginBottom: 6 },
  summaryText: { color: colors.textMuted, fontSize: 15 },
  card: {
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: radius.lg,
    padding: 16,
    backgroundColor: colors.surface,
    ...shadow.sm,
  },
  cardHeader: { flexDirection: "row", justifyContent: "space-between", alignItems: "center", marginBottom: 10 },
  qNumber: { fontWeight: "800", color: colors.textMuted },
  statusPill: {
    color: colors.white,
    paddingHorizontal: 11,
    paddingVertical: 3,
    borderRadius: 999,
    fontSize: 12,
    fontWeight: "700",
    overflow: "hidden",
    textTransform: "capitalize",
  },
  qText: { fontSize: 16, fontWeight: "700", color: colors.text, marginBottom: 12, lineHeight: 22 },
  answerRow: { flexDirection: "row", marginBottom: 5 },
  answerLabel: { color: colors.textMuted },
  answerValue: { fontWeight: "700" },
  explanationWrap: { marginTop: 12 },
  explanationToggle: { color: colors.brandDark, fontWeight: "700" },
  explanationText: {
    marginTop: 10,
    color: colors.text,
    lineHeight: 21,
    backgroundColor: colors.brandSoft,
    borderWidth: 1,
    borderColor: colors.brandSoftBorder,
    padding: 12,
    borderRadius: radius.md,
  },
  retryButton: { backgroundColor: colors.brand, borderRadius: radius.md, paddingVertical: 11, paddingHorizontal: 18, ...shadow.brand },
  retryText: { color: colors.white, fontWeight: "700" },
  error: { color: colors.danger },
});
