import { useState } from "react";
import { StyleSheet, Text, View } from "react-native";
import { Link, useRouter } from "expo-router";
import { requestPasswordReset } from "../../lib/auth";
import { Field, LogoMark, PrimaryButton } from "../../components/ui";
import { colors, radius, shadow, spacing } from "../../lib/theme";

export default function ForgotPasswordScreen() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [sent, setSent] = useState(false);

  async function handleSubmit() {
    if (!email.trim()) {
      setError("Enter the email you signed up with.");
      return;
    }
    setError(null);
    setLoading(true);
    try {
      await requestPasswordReset(email.trim().toLowerCase());
      setSent(true);
    } catch (err) {
      // The backend intentionally never reveals whether an account exists,
      // so any failure here is a genuine network/server error.
      setError(err instanceof Error ? err.message : "Could not send reset email");
    } finally {
      setLoading(false);
    }
  }

  return (
    <View style={styles.container}>
      <View style={styles.card}>
        <View style={styles.brandRow}>
          <LogoMark size={40} />
          <Text style={styles.brand}>StudyApp</Text>
        </View>

        {sent ? (
          <>
            <Text style={styles.title}>Check your email</Text>
            <Text style={styles.subtitle}>
              If an account exists for {email.trim()}, we&apos;ve sent a password reset link to it.
              Open the link, copy the reset code from it, then continue below.
            </Text>
            <PrimaryButton
              title="I have my reset code"
              onPress={() => router.push({ pathname: "/auth/reset-password", params: { email: email.trim() } })}
            />
          </>
        ) : (
          <>
            <Text style={styles.title}>Forgot password?</Text>
            <Text style={styles.subtitle}>
              Enter your account email and we&apos;ll send you a link to reset your password.
            </Text>

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

              {error ? <Text style={styles.error}>{error}</Text> : null}

              <PrimaryButton title="Send reset link" onPress={() => void handleSubmit()} loading={loading} />
            </View>
          </>
        )}

        <Link href="/auth/login" style={styles.link}>
          Back to sign in
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
  subtitle: { fontSize: 15, color: colors.textMuted, marginBottom: spacing.xl, lineHeight: 21 },
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
});
