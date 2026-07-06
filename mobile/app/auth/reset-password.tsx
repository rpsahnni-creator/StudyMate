import { useState } from "react";
import { ScrollView, StyleSheet, Text, View } from "react-native";
import { Link, useLocalSearchParams, useRouter } from "expo-router";
import { resetPassword } from "../../lib/auth";
import { Field, LogoMark, PasswordField, PrimaryButton } from "../../components/ui";
import { colors, radius, shadow, spacing } from "../../lib/theme";

export default function ResetPasswordScreen() {
  const router = useRouter();
  const params = useLocalSearchParams<{ token?: string }>();

  const [token, setToken] = useState(params.token ?? "");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [done, setDone] = useState(false);

  async function handleSubmit() {
    if (!token.trim()) {
      setError("Paste the reset code from your email.");
      return;
    }
    if (newPassword.length < 8) {
      setError("Password must be at least 8 characters.");
      return;
    }
    if (newPassword !== confirmPassword) {
      setError("Passwords do not match.");
      return;
    }
    setError(null);
    setLoading(true);
    try {
      await resetPassword(token.trim(), newPassword, confirmPassword);
      setDone(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not reset password");
    } finally {
      setLoading(false);
    }
  }

  return (
    <ScrollView style={styles.screen} contentContainerStyle={styles.container}>
      <View style={styles.card}>
        <View style={styles.brandRow}>
          <LogoMark size={40} />
          <Text style={styles.brand}>StudyApp</Text>
        </View>

        {done ? (
          <>
            <Text style={styles.title}>Password updated</Text>
            <Text style={styles.subtitle}>You can now sign in with your new password.</Text>
            <PrimaryButton title="Go to sign in" onPress={() => router.replace("/auth/login")} />
          </>
        ) : (
          <>
            <Text style={styles.title}>Reset password</Text>
            <Text style={styles.subtitle}>
              Open the reset email on any device, copy the code from the link (the part after{" "}
              <Text style={styles.code}>?token=</Text>), and paste it below.
            </Text>

            <View style={styles.form}>
              <Field
                label="Reset code"
                placeholder="Paste the token from your email"
                autoCapitalize="none"
                autoCorrect={false}
                value={token}
                onChangeText={setToken}
              />
              <PasswordField
                label="New password"
                placeholder="At least 8 characters"
                autoComplete="new-password"
                value={newPassword}
                onChangeText={setNewPassword}
              />
              <PasswordField
                label="Confirm new password"
                placeholder="Re-enter new password"
                autoComplete="new-password"
                value={confirmPassword}
                onChangeText={setConfirmPassword}
              />

              {error ? <Text style={styles.error}>{error}</Text> : null}

              <PrimaryButton title="Reset password" onPress={() => void handleSubmit()} loading={loading} />
            </View>
          </>
        )}

        <Link href="/auth/login" style={styles.link}>
          Back to sign in
        </Link>
      </View>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: colors.bg },
  container: { padding: spacing.xl, justifyContent: "center", flexGrow: 1 },
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
  code: { fontWeight: "800", color: colors.text },
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
