import { StyleSheet } from "react-native";
import { radius, shadow, spacing } from "./theme";

/** Shared layout + glass surfaces for sky-background tab screens */
export const skyScreen = StyleSheet.create({
  screen: { flex: 1 },
  content: { padding: spacing.xl, paddingBottom: 40, gap: spacing.lg },
  contentFlush: { paddingBottom: 40 },
  center: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: 12,
    padding: 24,
  },
  glass: {
    backgroundColor: "rgba(255, 255, 255, 0.92)",
    borderColor: "rgba(255, 255, 255, 0.95)",
  },
  heroCard: {
    backgroundColor: "rgba(255, 255, 255, 0.9)",
    borderRadius: radius.xl,
    borderWidth: 1,
    borderColor: "rgba(255, 255, 255, 0.95)",
    padding: spacing.lg,
    gap: spacing.md,
    ...shadow.md,
  },
  title: { fontSize: 28, fontWeight: "800", color: "#0F172A", letterSpacing: -0.4 },
  lead: { color: "#64748B", fontSize: 14, lineHeight: 20 },
  sectionTitle: { fontSize: 16, fontWeight: "800", color: "#0F172A", marginTop: 4, marginBottom: 8 },
});
