import { Pressable, ScrollView, StyleSheet, Text, View } from "react-native";
import { useRouter } from "expo-router";
import { Ionicons } from "@expo/vector-icons";
import { useAuth } from "../../hooks/useAuth";
import { useFeatureFlags } from "../../hooks/useFeatureFlags";
import { Card, LogoMark, PrimaryButton, SecondaryButton } from "../../components/ui";
import { colors, radius, shadow, spacing } from "../../lib/theme";

const ACTIONS = [
  {
    key: "scan",
    icon: "scan-circle-outline" as const,
    title: "Scan Chapter",
    desc: "Capture questions and build a quiz",
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
  const { logout } = useAuth();
  const flags = useFeatureFlags();

  async function handleLogout() {
    await logout();
    router.replace("/auth/login");
  }

  return (
    <ScrollView style={styles.screen} contentContainerStyle={styles.content}>
      <View style={styles.headerRow}>
        <LogoMark size={44} />
        <Text style={styles.brand}>StudyApp</Text>
      </View>

      <View style={styles.hero}>
        <Text style={styles.heroTitle}>Scan. Practice. Improve.</Text>
        <Text style={styles.heroSub}>
          Turn NCERT & state-board chapters into smart AI quizzes and track your progress.
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

      <SecondaryButton title="Sign out" onPress={handleLogout} style={styles.signOut} />
    </ScrollView>
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
  screen: { flex: 1, backgroundColor: colors.bg },
  content: { padding: spacing.xl, gap: spacing.xl, paddingBottom: 40 },
  headerRow: { flexDirection: "row", alignItems: "center", gap: 12 },
  brand: { fontSize: 22, fontWeight: "800", color: colors.text },
  hero: {
    backgroundColor: colors.brandSoft,
    borderRadius: radius.xl,
    borderWidth: 1,
    borderColor: colors.brandSoftBorder,
    padding: spacing.xl,
    gap: spacing.md,
  },
  heroTitle: { fontSize: 26, fontWeight: "800", color: colors.text },
  heroSub: { fontSize: 15, color: colors.textMuted, lineHeight: 22 },
  heroBtn: { marginTop: spacing.sm },
  grid: { flexDirection: "row", gap: spacing.md },
  actionCard: {
    flex: 1,
    backgroundColor: colors.surface,
    borderRadius: radius.lg,
    borderWidth: 1,
    borderColor: colors.border,
    padding: spacing.lg,
    gap: 8,
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
  actionTitle: { fontSize: 16, fontWeight: "700", color: colors.text, marginTop: 4 },
  actionDesc: { fontSize: 13, color: colors.textMuted, lineHeight: 18 },
  flagsCard: { gap: spacing.md },
  flagLabel: { fontWeight: "800", color: colors.text, fontSize: 15 },
  flagRow: { flexDirection: "row", alignItems: "center", justifyContent: "space-between" },
  flagName: { color: colors.text, fontSize: 14 },
  statusWrap: { flexDirection: "row", alignItems: "center", gap: 6 },
  dot: { width: 8, height: 8, borderRadius: 999 },
  statusText: { fontSize: 12, fontWeight: "700" },
  signOut: { marginTop: spacing.sm },
});
