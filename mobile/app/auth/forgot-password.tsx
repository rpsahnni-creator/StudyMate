import { useState } from "react";
import { StyleSheet, Text, View } from "react-native";
import { Link, useRouter } from "expo-router";
import { requestPasswordReset } from "../../lib/auth";
import {
  AuthScreenLayout,
  authPalette,
  authPrimaryBtnStyle,
  authStyles,
} from "../../components/AuthScreenLayout";
import { Field, PrimaryButton } from "../../components/ui";
import { spacing } from "../../lib/theme";

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
      setError(err instanceof Error ? err.message : "Could not send reset email");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthScreenLayout sectionLabel={sent ? "Check your email" : "Reset your password"}>
      {sent ? (
        <View style={styles.sentBlock}>
          <Text style={styles.sentText}>
            If an account exists for {email.trim()}, we&apos;ve sent a password reset link to it.
            Open the link, copy the reset code from it, then continue below.
          </Text>
          <PrimaryButton
            title="I have my reset code"
            onPress={() => router.push({ pathname: "/auth/reset-password", params: { email: email.trim() } })}
            style={authPrimaryBtnStyle}
            labelColor={authPalette.ink}
          />
        </View>
      ) : (
        <View style={styles.form}>
          <Text style={styles.hint}>
            Enter your account email and we&apos;ll send you a link to reset your password.
          </Text>
          <Field
            label="Email"
            placeholder="you@example.com"
            autoCapitalize="none"
            keyboardType="email-address"
            autoComplete="email"
            value={email}
            onChangeText={setEmail}
          />
          {error ? <Text style={authStyles.error}>{error}</Text> : null}
          <PrimaryButton
            title="Send reset link"
            onPress={() => void handleSubmit()}
            loading={loading}
            style={authPrimaryBtnStyle}
            labelColor={authPalette.ink}
          />
        </View>
      )}

      <Link href="/auth/login" style={authStyles.link}>
        Back to sign in
      </Link>
    </AuthScreenLayout>
  );
}

const styles = StyleSheet.create({
  form: { gap: spacing.lg },
  sentBlock: { gap: spacing.lg },
  hint: {
    fontSize: 14,
    color: authPalette.muted,
    lineHeight: 21,
  },
  sentText: {
    fontSize: 14,
    color: authPalette.muted,
    lineHeight: 21,
  },
});
