import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import { useRouter } from "expo-router";
import { Ionicons } from "@expo/vector-icons";
import { useFeatureFlags } from "../../hooks/useFeatureFlags";
import { SkyBackground } from "../../components/SkyBackground";
import { StudentProfileHeader } from "../../components/StudentProfileHeader";
import { Card, PrimaryButton } from "../../components/ui";
import { colors, radius, shadow, spacing } from "../../lib/theme";

const ACTIONS = [
  {
    key: "scan",
    icon: "scan-circle-outline" as const,
    title: "Scan Chapter",
    desc: "Capture lessons and create quizzes",
    color: colors.brand,
    route: "/(tabs)/scan" as const,
  },
  {
    key: "reports",
    icon: "bar-chart-outline" as const,
    title: "My Reports",
    desc: "Track scores and streaks",
    color: "#0ea5e9",
    route: "/(tabs)/reports" as const,
  },
];

export default function HomeScreen() {
  const router = useRouter();
  const flags = useFeatureFlags();

  return (
    <SkyBackground>
      <ScrollView style={styles.screen} contentContainerStyle={styles.content}>
        <StudentProfileHeader />

        <View style={styles.hero}>
          <Text style={styles.heroTitle}>Scan → Practice → Grow</Text>
          <Text style={styles.heroSub}>
            Smart AI quizzes and track your progress.
          </Text>
          <PrimaryButton
            title="Start scanning →"
            onPress={() => router.push("/(tabs)/scan")}
            style={styles.heroBtn}
          />
        </View>

        <View style={styles.grid}>
          {ACTIONS.map((a) => (
            <Pressable
              key={a.key}
              style={({ pressed }: { pressed: boolean }) => [styles.actionCard, pressed ? styles.pressed : null]}
              onPress={() => router.push(a.route)}
            >
              <View style={[styles.actionIcon, { backgroundColor: a.color }]}>
                <Ionicons name={a.icon} size={24} color={colors.white} />
              </View>
              <Text style={styles.actionTitle}>{a.title}</Text>
              <Text style={styles.actionDesc}>{a.desc}</Text>
            </Pressable>
          ))}
        </View>

        <Card style={styles.flagsCard}>
          <Text style={styles.flagLabel}>Active modules</Text>
          <View style={styles.flagRow}>
            <Text style={styles.flagName}>Scan & Quiz</Text>
            <StatusDot on={flags.scan_quiz_module} />
          </View>
          <View style={styles.flagRow}>
            <Text style={styles.flagName}>Career Goals</Text>
            <StatusDot on={flags.career_goals_module} />
          </View>
        </Card>
      </ScrollView>
    </SkyBackground>
  );
}

function StatusDot({ on }: { on: boolean }) {
  return (
    <View style={styles.statusWrap}>
      <View
        style={[styles.dot, { backgroundColor: on ? colors.success : colors.textSubtle }]}
      />
      <Text style={[styles.statusText, { color: on ? colors.success : colors.textSubtle }]}>
        {on ? "ON" : "OFF"}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1 },
  content: { padding: spacing.xl, gap: spacing.xl, paddingBottom: 40 },
  hero: {
    backgroundColor: "rgba(255, 255, 255, 0.9)",
    borderRadius: radius.xl,
    borderWidth: 1,
    borderColor: "rgba(255, 255, 255, 0.95)",
    padding: spacing.xl,
    gap: spacing.md,
    alignItems: "center",
    ...shadow.sm,
  },
  heroTitle: { fontSize: 26, fontWeight: "800", color: colors.text, textAlign: "center" },
  heroSub: { fontSize: 15, color: colors.textMuted, lineHeight: 22, textAlign: "center" },
  heroBtn: { marginTop: spacing.sm, alignSelf: "stretch" },
  grid: { flexDirection: "row", gap: spacing.md },
  actionCard: {
    flex: 1,
    backgroundColor: "rgba(255, 255, 255, 0.92)",
    borderRadius: radius.lg,
    borderWidth: 1,
    borderColor: "rgba(255, 255, 255, 0.95)",
    padding: spacing.lg,
    gap: 8,
    alignItems: "center",
    ...shadow.sm,
  },
  pressed: { opacity: 0.9, transform: [{ scale: 0.98 }] },
  actionIcon: {
    width: 46,
    height: 46,
    borderRadius: 13,
    alignItems: "center",
    justifyContent: "center",
    ...shadow.md,
  },
  actionTitle: { fontSize: 16, fontWeight: "700", color: colors.text, marginTop: 4, textAlign: "center" },
  actionDesc: { fontSize: 13, color: colors.textMuted, lineHeight: 18, textAlign: "center" },
  flagsCard: {
    gap: spacing.md,
    backgroundColor: "rgba(255, 255, 255, 0.9)",
    borderColor: "rgba(255, 255, 255, 0.95)",
  },
  flagLabel: { fontWeight: "800", color: colors.text, fontSize: 15 },
  flagRow: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  flagName: { color: colors.text, fontSize: 14 },
  statusWrap: { flexDirection: "row", alignItems: "center", gap: 6 },
  dot: { width: 8, height: 8, borderRadius: 999 },
  statusText: { fontSize: 12, fontWeight: "700" },
});
