import type { ReactNode } from "react";
import { ScrollView, StyleSheet, Text, View, type ViewProps } from "react-native";
import { StatusBar } from "expo-status-bar";
import { SafeAreaView } from "react-native-safe-area-context";
import { LogoMark } from "./ui";
import { radius, spacing } from "../lib/theme";

export const authPalette = {
  night: "#050508",
  white: "#FFFFFF",
  ink: "#0F172A",
  muted: "#94A3B8",
  gold: "#F0B429",
  goldDark: "#D49A12",
  brandMuted: "#9EB4C8",
};

export const authPrimaryBtnStyle = {
  backgroundColor: authPalette.gold,
  shadowColor: authPalette.goldDark,
  borderRadius: 999,
};

type AuthScreenLayoutProps = ViewProps & {
  sectionLabel: string;
  children: ReactNode;
};

export function AuthScreenLayout({ sectionLabel, children, style, ...rest }: AuthScreenLayoutProps) {
  return (
    <View style={[styles.root, style]} {...rest}>
      <View style={styles.glowCyan} pointerEvents="none" />
      <View style={styles.glowOrange} pointerEvents="none" />
      <View style={styles.glowViolet} pointerEvents="none" />

      <SafeAreaView style={styles.safe} edges={["top", "bottom"]}>
        <StatusBar style="light" />
        <ScrollView
          contentContainerStyle={styles.scroll}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}
        >
          <View style={styles.hero}>
            <View style={styles.logoWrap}>
              <LogoMark width={280} height={100} />
            </View>
            <Text style={styles.brandName}>Kiji Technology</Text>
            <Text style={styles.heroTitle}>
              <Text style={styles.heroStudy}>Study</Text>
              <Text style={styles.heroMate}>Mate</Text>
            </Text>
            <Text style={styles.heroSubtitle}>Learn Smarter Every Day</Text>
          </View>

          <View style={styles.formSection}>
            <Text style={styles.sectionLabel}>{sectionLabel}</Text>
            <View style={styles.card}>{children}</View>
          </View>
        </ScrollView>
      </SafeAreaView>
    </View>
  );
}

export const authStyles = StyleSheet.create({
  form: { gap: spacing.lg },
  error: {
    color: "#dc2626",
    backgroundColor: "#fef2f2",
    padding: 11,
    borderRadius: radius.md,
    fontWeight: "500",
  },
  link: {
    marginTop: spacing.xl,
    textAlign: "center",
    color: authPalette.ink,
    fontWeight: "700",
    fontSize: 14,
    textDecorationLine: "underline",
  },
  mutedLink: {
    textAlign: "center",
    color: authPalette.muted,
    fontWeight: "600",
    fontSize: 13.5,
  },
});

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: authPalette.night,
    overflow: "hidden",
  },
  glowCyan: {
    position: "absolute",
    width: 340,
    height: 340,
    borderRadius: 170,
    backgroundColor: "rgba(34, 211, 238, 0.42)",
    bottom: -40,
    left: -90,
  },
  glowOrange: {
    position: "absolute",
    width: 300,
    height: 300,
    borderRadius: 150,
    backgroundColor: "rgba(32, 178, 170, 0.34)",
    bottom: 20,
    right: -70,
  },
  glowViolet: {
    position: "absolute",
    width: 220,
    height: 220,
    borderRadius: 110,
    backgroundColor: "rgba(99, 102, 241, 0.22)",
    top: -40,
    right: -30,
  },
  safe: {
    flex: 1,
    backgroundColor: "transparent",
  },
  scroll: {
    flexGrow: 1,
    justifyContent: "center",
    padding: spacing.xl,
    paddingVertical: spacing.xxl,
  },
  hero: {
    alignItems: "center",
    marginBottom: spacing.xl,
  },
  logoWrap: {
    marginBottom: spacing.sm,
  },
  brandName: {
    fontSize: 15,
    fontWeight: "600",
    color: authPalette.brandMuted,
    textAlign: "center",
    letterSpacing: 0.5,
    marginBottom: spacing.md,
  },
  heroTitle: {
    fontSize: 38,
    textAlign: "center",
    marginBottom: 10,
    lineHeight: 42,
  },
  heroStudy: {
    color: authPalette.white,
    fontWeight: "300",
    letterSpacing: 2,
  },
  heroMate: {
    color: authPalette.gold,
    fontWeight: "800",
    fontStyle: "italic",
    letterSpacing: -0.5,
    textShadowColor: "rgba(240, 180, 41, 0.35)",
    textShadowOffset: { width: 0, height: 2 },
    textShadowRadius: 10,
  },
  heroSubtitle: {
    fontSize: 15,
    color: "rgba(255, 255, 255, 0.72)",
    textAlign: "center",
    lineHeight: 22,
    maxWidth: 320,
  },
  formSection: {
    width: "100%",
    maxWidth: 440,
    alignSelf: "center",
  },
  sectionLabel: {
    fontSize: 13,
    fontWeight: "700",
    color: "rgba(255, 255, 255, 0.55)",
    marginBottom: 12,
    marginLeft: 4,
    letterSpacing: 0.4,
    textTransform: "uppercase",
  },
  card: {
    backgroundColor: authPalette.white,
    borderRadius: 26,
    padding: spacing.xl,
    shadowColor: "#000",
    shadowOpacity: 0.35,
    shadowRadius: 30,
    shadowOffset: { width: 0, height: 16 },
    elevation: 8,
  },
});
