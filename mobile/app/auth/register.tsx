import { useState } from "react";
import { Text, View } from "react-native";
import { Link, useRouter } from "expo-router";
import { API_URL, parseApiError } from "../../lib/auth";
import { useAuth } from "../../hooks/useAuth";
import {
  AuthScreenLayout,
  authPalette,
  authPrimaryBtnStyle,
  authStyles,
} from "../../components/AuthScreenLayout";
import { Field, PasswordField, PrimaryButton } from "../../components/ui";

export default function RegisterScreen() {
  const router = useRouter();
  const { login } = useAuth();

  const [name, setName] = useState("");
  const [classLevel, setClassLevel] = useState("");
  const [email, setEmail] = useState("");
  const [mobile, setMobile] = useState("");
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");
  const [otp, setOtp] = useState("");
  const [otpSent, setOtpSent] = useState(false);
  const [emailVerified, setEmailVerified] = useState(false);
  const [verificationToken, setVerificationToken] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [info, setInfo] = useState<string | null>(null);
  const [sendingOtp, setSendingOtp] = useState(false);
  const [verifyingOtp, setVerifyingOtp] = useState(false);
  const [loading, setLoading] = useState(false);

  const normalizedEmail = email.trim().toLowerCase();

  async function handleSendOtp() {
    setError(null);
    setInfo(null);
    if (!normalizedEmail) {
      setError("Email is required");
      return;
    }

    setSendingOtp(true);
    try {
      const res = await fetch(`${API_URL}/auth/register/send-otp`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: normalizedEmail }),
      });
      if (!res.ok) {
        throw new Error(await parseApiError(res));
      }
      const data = (await res.json()) as { message?: string; dev_otp?: string };
      setOtpSent(true);
      setEmailVerified(false);
      setVerificationToken(null);
      if (data.dev_otp) {
        setOtp(data.dev_otp);
        setInfo(`Dev mode: OTP is ${data.dev_otp} (inbox mein nahi jayega — stub email)`);
      } else {
        setInfo("Verification code sent to your email.");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to send code");
    } finally {
      setSendingOtp(false);
    }
  }

  async function handleVerifyOtp() {
    setError(null);
    setInfo(null);
    if (!otp.trim()) {
      setError("Enter the 6-digit code");
      return;
    }

    setVerifyingOtp(true);
    try {
      const res = await fetch(`${API_URL}/auth/register/verify-otp`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: normalizedEmail, otp: otp.trim() }),
      });
      if (!res.ok) {
        throw new Error(await parseApiError(res));
      }
      const data = (await res.json()) as { verification_token: string };
      setVerificationToken(data.verification_token);
      setEmailVerified(true);
      setInfo("Email verified. You can create your account.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Verification failed");
    } finally {
      setVerifyingOtp(false);
    }
  }

  async function handleRegister() {
    setError(null);
    setInfo(null);

    if (!emailVerified || !verificationToken) {
      setError("Please verify your email first");
      return;
    }
    if (password !== passwordConfirm) {
      setError("Passwords do not match");
      return;
    }

    setLoading(true);
    try {
      const registerRes = await fetch(`${API_URL}/auth/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          verification_token: verificationToken,
          name: name.trim(),
          class: classLevel.trim(),
          email: normalizedEmail,
          mobile: mobile.trim(),
          password,
          password_confirm: passwordConfirm,
          accept_terms: true,
        }),
      });

      if (!registerRes.ok) {
        throw new Error(await parseApiError(registerRes));
      }

      await login(normalizedEmail, password);
      router.replace("/(tabs)");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Registration failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthScreenLayout sectionLabel="Create your account">
      <View style={authStyles.form}>
        <Field
          label="Name"
          placeholder="Your full name"
          autoComplete="name"
          value={name}
          onChangeText={setName}
        />
        <Field
          label="Class"
          placeholder="e.g. 10, 12, B.Tech 2nd year"
          value={classLevel}
          onChangeText={setClassLevel}
        />
        <Field
          label="Email"
          placeholder="you@example.com"
          autoCapitalize="none"
          keyboardType="email-address"
          autoComplete="email"
          value={email}
          onChangeText={(value) => {
            setEmail(value);
            setOtpSent(false);
            setEmailVerified(false);
            setVerificationToken(null);
            setOtp("");
          }}
        />
        <Field
          label="Mobile number"
          placeholder="10-digit mobile"
          keyboardType="phone-pad"
          autoComplete="tel"
          value={mobile}
          onChangeText={setMobile}
        />

        <View style={{ gap: 8 }}>
          <PrimaryButton
            title={otpSent ? "Resend code" : "Send verification code"}
            onPress={handleSendOtp}
            loading={sendingOtp}
            style={[authPrimaryBtnStyle, { opacity: emailVerified ? 0.6 : 1 }]}
            labelColor={authPalette.ink}
            disabled={emailVerified}
          />
          {otpSent && !emailVerified ? (
            <>
              <Field
                label="Email verification code"
                placeholder="6-digit code"
                keyboardType="number-pad"
                value={otp}
                onChangeText={setOtp}
              />
              <PrimaryButton
                title="Verify email"
                onPress={handleVerifyOtp}
                loading={verifyingOtp}
                style={authPrimaryBtnStyle}
                labelColor={authPalette.ink}
              />
            </>
          ) : null}
          {emailVerified ? (
            <Text style={{ color: "#16a34a", fontSize: 14, fontWeight: "600" }}>
              Email verified
            </Text>
          ) : null}
        </View>

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

        {info ? <Text style={{ color: authPalette.muted, fontSize: 14 }}>{info}</Text> : null}
        {error ? <Text style={authStyles.error}>{error}</Text> : null}

        <PrimaryButton
          title="Create account"
          onPress={handleRegister}
          loading={loading}
          style={[authPrimaryBtnStyle, { opacity: emailVerified ? 1 : 0.5 }]}
          labelColor={authPalette.ink}
          disabled={!emailVerified}
        />

        <Link href="/auth/login" style={authStyles.link}>
          Already have an account? Sign in
        </Link>
      </View>
    </AuthScreenLayout>
  );
}
