import { useState } from "react";
import { ScrollView, StyleSheet, Text, View } from "react-native";
import { Link, useRouter } from "expo-router";
import { StatusBar } from "expo-status-bar";
import { SafeAreaView } from "react-native-safe-area-context";
import { useAuth } from "../../hooks/useAuth";
import { Field, LogoMark, PasswordField, PrimaryButton } from "../../components/ui";
import { radius, spacing } from "../../lib/theme";

const palette = {
  night: "#050508",
  white: "#FFFFFF",
  ink: "#0F172A",
  muted: "#94A3B8",
  cyan: "#22D3EE",
  gold: "#F0B429",
  goldDark: "#D49A12",
  seaGreen: "#20B2AA",
  brandMuted: "#9EB4C8",
  glass: "rgba(255, 255, 255, 0.07)",
  glassBorder: "rgba(255, 255, 255, 0.12)",
};

export default function LoginScreen() {
  const router = useRouter();
  const { login } = useAuth();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function submitLogin(loginEmail: string, loginPassword: string) {
    setError(null);
    setLoading(true);
    try {
      await login(loginEmail, loginPassword);
      router.replace("/(tabs)");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  function handleLogin() {
    void submitLogin(email, password);
  }

  return (
    <View style={styles.root}>
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
            <Text style={styles.sectionLabel}>Sign in to your account</Text>

            <View style={styles.card}>
              <View style={styles.form}>
                <Field
                  label="Email"
                  placeholder="you@example.com"
                  autoCapitalize="none"
                  keyboardType="email-address"
                  autoComplete="email"
                  value={email}
                  onChangeText={setEmail}
                />
                <PasswordField
                  label="Password"
                  placeholder="••••••••"
                  autoComplete="password"
                  value={password}
                  onChangeText={setPassword}
                />

                {error ? <Text style={styles.error}>{error}</Text> : null}

                <PrimaryButton
                  title="Sign in"
                  onPress={handleLogin}
                  loading={loading}
                  style={styles.signInBtn}
                  labelColor={palette.ink}
                />

                <Link href="/auth/forgot-password" style={styles.forgotLink}>
                  Forgot password?
                </Link>
              </View>

              <Link href="/auth/register" style={styles.link}>
                No account? Create one
              </Link>
            </View>
          </View>
        </ScrollView>
      </SafeAreaView>
    </View>
  );
}

const styles = StyleSheet.create({
  root: {
    flex: 1,
    backgroundColor: palette.night,
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
    color: palette.brandMuted,
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
    color: palette.white,
    fontWeight: "300",
    letterSpacing: 2,
  },
  heroMate: {
    color: palette.gold,
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
    backgroundColor: palette.white,
    borderRadius: 26,
    padding: spacing.xl,
    shadowColor: "#000",
    shadowOpacity: 0.35,
    shadowRadius: 30,
    shadowOffset: { width: 0, height: 16 },
    elevation: 8,
  },
  form: { gap: spacing.lg },
  signInBtn: {
    backgroundColor: palette.gold,
    shadowColor: palette.goldDark,
    borderRadius: 999,
  },
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
    color: palette.ink,
    fontWeight: "700",
    fontSize: 14,
    textDecorationLine: "underline",
  },
  forgotLink: {
    textAlign: "center",
    color: palette.muted,
    fontWeight: "600",
    fontSize: 13.5,
  },
});
