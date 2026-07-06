import { useState } from "react";
import { StyleSheet, Text, View } from "react-native";
import { Link, useRouter } from "expo-router";
import { useAuth } from "../../hooks/useAuth";
import { DEMO_EMAIL, DEMO_PASSWORD } from "../../lib/demoAccount";
import { API_URL } from "../../lib/config";
import { Field, LogoMark, PasswordField, PrimaryButton, SecondaryButton } from "../../components/ui";
import { colors, radius, shadow, spacing } from "../../lib/theme";

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
      router.replace("/(tabs)/scan");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  function handleLogin() {
    void submitLogin(email, password);
  }

  function handleDemoLogin() {
    setEmail(DEMO_EMAIL);
    setPassword(DEMO_PASSWORD);
    void submitLogin(DEMO_EMAIL, DEMO_PASSWORD);
  }

  return (
    <View style={styles.container}>
      <View style={styles.card}>
        <View style={styles.brandRow}>
          <LogoMark size={40} />
          <Text style={styles.brand}>StudyApp</Text>
        </View>
        <Text style={styles.title}>Welcome back</Text>
        <Text style={styles.subtitle}>Sign in to scan chapters and generate quizzes.</Text>

        <View style={styles.form}>
          <Field
            label="Email"
            placeholder="demo@123.com"
            autoCapitalize="none"
            keyboardType="email-address"
            autoComplete="email"
            value={email}
            onChangeText={setEmail}
          />
          <PasswordField
            label="Password"
            placeholder="Demo@123"
            autoComplete="password"
            value={password}
            onChangeText={setPassword}
          />

          {error ? <Text style={styles.error}>{error}</Text> : null}

          <PrimaryButton title="Sign in" onPress={handleLogin} loading={loading} />

          {__DEV__ ? (
            <>
              <SecondaryButton title="Demo account se login" onPress={handleDemoLogin} disabled={loading} />
              <Text style={styles.demoHint}>
                Demo: {DEMO_EMAIL} / {DEMO_PASSWORD}
                {"\n"}Server: {API_URL}
              </Text>
            </>
          ) : null}

          <Link href="/auth/forgot-password" style={styles.forgotLink}>
            Forgot password?
          </Link>
        </View>

        <Link href="/auth/register" style={styles.link}>
          No account? Create one
        </Link>
      </View>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    padding: spacing.xl,
    justifyContent: "center",
    backgroundColor: colors.bg,
  },
  card: {
    backgroundColor: colors.surface,
    borderRadius: radius.xl,
    borderWidth: 1,
    borderColor: colors.border,
    padding: spacing.xl,
    ...shadow.md,
  },
  brandRow: { flexDirection: "row", alignItems: "center", gap: 10, marginBottom: spacing.xl },
  brand: { fontSize: 19, fontWeight: "800", color: colors.text },
  title: { fontSize: 27, fontWeight: "800", color: colors.text, marginBottom: 6 },
  subtitle: { fontSize: 15, color: colors.textMuted, marginBottom: spacing.xl },
  form: { gap: spacing.lg },
  error: {
    color: colors.danger,
    backgroundColor: colors.dangerBg,
    padding: 11,
    borderRadius: radius.md,
    fontWeight: "500",
  },
  link: {
    marginTop: spacing.xl,
    textAlign: "center",
    color: colors.brandDark,
    fontWeight: "700",
    fontSize: 14,
  },
  forgotLink: {
    textAlign: "center",
    color: colors.textMuted,
    fontWeight: "600",
    fontSize: 13.5,
  },
  demoHint: {
    fontSize: 12,
    color: colors.textSubtle,
    textAlign: "center",
    lineHeight: 17,
  },
});
