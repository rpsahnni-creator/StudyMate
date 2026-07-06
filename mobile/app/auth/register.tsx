import { useState } from "react";
import { ScrollView, StyleSheet, Text, View } from "react-native";
import { Link, useRouter } from "expo-router";
import { API_URL, parseApiError } from "../../lib/auth";
import { useAuth } from "../../hooks/useAuth";
import { Field, LogoMark, PasswordField, PrimaryButton } from "../../components/ui";
import { colors, radius, shadow, spacing } from "../../lib/theme";

export default function RegisterScreen() {
  const router = useRouter();
  const { login } = useAuth();

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleRegister() {
    setError(null);

    if (password !== passwordConfirm) {
      setError("Passwords do not match");
      return;
    }

    setLoading(true);
    try {
      const normalizedEmail = email.trim().toLowerCase();
      const registerRes = await fetch(`${API_URL}/auth/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          name: name.trim(),
          email: normalizedEmail,
          password,
          password_confirm: passwordConfirm,
          accept_terms: true,
        }),
      });

      if (!registerRes.ok) {
        throw new Error(await parseApiError(registerRes));
      }

      await login(normalizedEmail, password);
      router.replace("/(tabs)/scan");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Registration failed");
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
        <Text style={styles.title}>Create account</Text>
        <Text style={styles.subtitle}>Join StudyApp to start learning.</Text>

        <View style={styles.form}>
          <Field label="Name" placeholder="Your name" autoComplete="name" value={name} onChangeText={setName} />
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
            placeholder="At least 8 characters"
            autoComplete="new-password"
            value={password}
            onChangeText={setPassword}
          />
          <PasswordField
            label="Confirm password"
            placeholder="Re-enter password"
            autoComplete="new-password"
            value={passwordConfirm}
            onChangeText={setPasswordConfirm}
          />

          {error ? <Text style={styles.error}>{error}</Text> : null}

          <PrimaryButton title="Create account" onPress={handleRegister} loading={loading} />
        </View>

        <Link href="/auth/login" style={styles.link}>
          Already have an account? Sign in
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
});
