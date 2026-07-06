"use client";

import { FormEvent, useState, type CSSProperties, type ReactNode } from "react";
import Link from "next/link";
import { Outfit } from "next/font/google";
import { useRouter } from "next/navigation";
import { KijiLogo } from "../../../components/KijiLogo";
import {
  API_URL,
  parseAuthError,
  persistSession,
  type TokenResponse,
} from "../../../lib/auth";

const titleFont = Outfit({
  subsets: ["latin"],
  weight: ["300", "800"],
});

const palette = {
  night: "#050508",
  white: "#FFFFFF",
  ink: "#0F172A",
  muted: "#64748B",
  gold: "#F0B429",
  brandMuted: "#9EB4C8",
};

function AuthScreenShell({
  sectionLabel,
  children,
}: {
  sectionLabel: string;
  children: ReactNode;
}) {
  return (
    <main style={shellStyles.main}>
      <div style={shellStyles.glowCyan} aria-hidden />
      <div style={shellStyles.glowOrange} aria-hidden />
      <div style={shellStyles.glowViolet} aria-hidden />

      <div style={shellStyles.content} className="animate-in">
        <div style={shellStyles.hero}>
          <Link href="/" style={shellStyles.logoWrap}>
            <KijiLogo width={280} height={100} className="kiji-logo-hero" priority />
          </Link>
          <p style={shellStyles.brandName}>Kiji Technology</p>
          <h1 className={titleFont.className} style={shellStyles.heroTitle}>
            <span style={shellStyles.heroStudy}>Study</span>
            <span style={shellStyles.heroMate}>Mate</span>
          </h1>
          <p style={shellStyles.heroSubtitle}>Learn Smarter Every Day</p>
        </div>

        <div style={shellStyles.formSection}>
          <p style={shellStyles.sectionLabel}>{sectionLabel}</p>
          <div style={shellStyles.card}>{children}</div>
        </div>
      </div>
    </main>
  );
}

export default function RegisterPage() {
  const router = useRouter();

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
        setError(await parseAuthError(res));
        return;
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
        setError(await parseAuthError(res));
        return;
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

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
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
    if (!acceptTerms) {
      setError("You must accept the terms of service");
      return;
    }

    setLoading(true);

    try {
      const registerRes = await fetch(`${API_URL}/auth/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          verification_token: verificationToken,
          name,
          class: classLevel,
          email: normalizedEmail,
          mobile,
          password,
          password_confirm: passwordConfirm,
          accept_terms: true,
        }),
      });

      if (!registerRes.ok) {
        setError(await parseAuthError(registerRes));
        return;
      }

      const loginRes = await fetch(`${API_URL}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: normalizedEmail, password }),
      });

      if (!loginRes.ok) {
        setError(await parseAuthError(loginRes));
        return;
      }

      const data = (await loginRes.json()) as TokenResponse;
      await persistSession(data);
      router.push("/scan");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Registration failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AuthScreenShell sectionLabel="Create your account">
      <form onSubmit={handleSubmit} style={formStyles.form}>
        <label style={formStyles.label}>
          Name
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
            autoComplete="name"
            minLength={2}
            placeholder="Your full name"
            style={formStyles.input}
          />
        </label>

        <label style={formStyles.label}>
          Class
          <input
            type="text"
            value={classLevel}
            onChange={(e) => setClassLevel(e.target.value)}
            required
            placeholder="e.g. 10, 12, B.Tech 2nd year"
            style={formStyles.input}
          />
        </label>

        <label style={formStyles.label}>
          Email
          <input
            type="email"
            value={email}
            onChange={(e) => {
              setEmail(e.target.value);
              setOtpSent(false);
              setEmailVerified(false);
              setVerificationToken(null);
              setOtp("");
            }}
            required
            autoComplete="email"
            placeholder="you@example.com"
            style={formStyles.input}
          />
        </label>

        <label style={formStyles.label}>
          Mobile number
          <input
            type="tel"
            value={mobile}
            onChange={(e) => setMobile(e.target.value)}
            required
            autoComplete="tel"
            placeholder="10-digit mobile"
            pattern="[0-9]{10}"
            style={formStyles.input}
          />
        </label>

        <div style={formStyles.otpBlock}>
          <button
            type="button"
            disabled={sendingOtp || emailVerified}
            onClick={() => void handleSendOtp()}
            className="btn-reset"
            style={{
              ...formStyles.secondaryButton,
              opacity: emailVerified ? 0.6 : 1,
            }}
          >
            {sendingOtp ? "Sending…" : otpSent ? "Resend code" : "Send verification code"}
          </button>

          {otpSent && !emailVerified ? (
            <>
              <label style={formStyles.label}>
                Email verification code
                <input
                  type="text"
                  inputMode="numeric"
                  value={otp}
                  onChange={(e) => setOtp(e.target.value)}
                  maxLength={6}
                  placeholder="6-digit code"
                  style={formStyles.input}
                />
              </label>
              <button
                type="button"
                disabled={verifyingOtp}
                onClick={() => void handleVerifyOtp()}
                className="btn-reset"
                style={formStyles.secondaryButton}
              >
                {verifyingOtp ? "Verifying…" : "Verify email"}
              </button>
            </>
          ) : null}

          {emailVerified ? (
            <p style={formStyles.success}>Email verified</p>
          ) : null}
        </div>

        <label style={formStyles.label}>
          Password
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            autoComplete="new-password"
            minLength={8}
            placeholder="At least 8 characters"
            style={formStyles.input}
          />
        </label>

        <label style={formStyles.label}>
          Confirm password
          <input
            type="password"
            value={passwordConfirm}
            onChange={(e) => setPasswordConfirm(e.target.value)}
            required
            autoComplete="new-password"
            minLength={8}
            placeholder="Re-enter password"
            style={formStyles.input}
          />
        </label>

        <label style={formStyles.checkboxLabel}>
          <input
            type="checkbox"
            checked={acceptTerms}
            onChange={(e) => setAcceptTerms(e.target.checked)}
          />
          <span>I accept the terms of service</span>
        </label>

        {info ? <p style={formStyles.info}>{info}</p> : null}
        {error ? <p style={formStyles.error}>{error}</p> : null}

        <button
          type="submit"
          disabled={loading || !emailVerified}
          className="btn-reset"
          style={{
            ...formStyles.button,
            opacity: emailVerified ? 1 : 0.5,
          }}
        >
          {loading ? "Creating account…" : "Create account"}
        </button>
      </form>

      <p style={formStyles.footer}>
        Already have an account?{" "}
        <Link href="/auth/login" style={formStyles.link}>
          Sign in
        </Link>
      </p>
    </AuthScreenShell>
  );
}

const shellStyles: Record<string, CSSProperties> = {
  main: {
    position: "relative",
    minHeight: "100vh",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    padding: "40px 24px",
    overflow: "hidden",
    background: palette.night,
  },
  glowCyan: {
    position: "absolute",
    width: 520,
    height: 520,
    borderRadius: "50%",
    background: "radial-gradient(circle, rgba(34, 211, 238, 0.5) 0%, rgba(34, 211, 238, 0) 68%)",
    filter: "blur(8px)",
    bottom: "-140px",
    left: "-120px",
    pointerEvents: "none",
  },
  glowOrange: {
    position: "absolute",
    width: 460,
    height: 460,
    borderRadius: "50%",
    background: "radial-gradient(circle, rgba(32, 178, 170, 0.42) 0%, rgba(32, 178, 170, 0) 68%)",
    filter: "blur(8px)",
    bottom: "-60px",
    right: "-100px",
    pointerEvents: "none",
  },
  glowViolet: {
    position: "absolute",
    width: 320,
    height: 320,
    borderRadius: "50%",
    background: "radial-gradient(circle, rgba(99, 102, 241, 0.28) 0%, rgba(99, 102, 241, 0) 70%)",
    filter: "blur(6px)",
    top: "-80px",
    right: "-40px",
    pointerEvents: "none",
  },
  content: {
    position: "relative",
    zIndex: 1,
    width: "100%",
    maxWidth: 440,
  },
  hero: {
    textAlign: "center",
    marginBottom: 28,
  },
  logoWrap: {
    display: "inline-flex",
    justifyContent: "center",
    alignItems: "center",
    marginBottom: 8,
    textDecoration: "none",
  },
  brandName: {
    margin: "0 0 14px",
    fontSize: 15,
    fontWeight: 600,
    color: palette.brandMuted,
    textAlign: "center",
    letterSpacing: "0.04em",
  },
  heroTitle: {
    margin: "0 0 10px",
    fontSize: 40,
    lineHeight: 1.1,
    textAlign: "center",
  },
  heroStudy: {
    fontWeight: 300,
    color: palette.white,
    letterSpacing: "0.12em",
  },
  heroMate: {
    fontWeight: 800,
    fontStyle: "italic",
    letterSpacing: "-0.02em",
    background: `linear-gradient(135deg, ${palette.gold} 0%, #FDE68A 45%, #22D3EE 100%)`,
    WebkitBackgroundClip: "text",
    WebkitTextFillColor: "transparent",
    backgroundClip: "text",
    filter: "drop-shadow(0 2px 12px rgba(240, 180, 41, 0.35))",
  },
  heroSubtitle: {
    margin: 0,
    color: "rgba(255, 255, 255, 0.72)",
    fontSize: 15,
    lineHeight: 1.55,
  },
  formSection: {
    width: "100%",
  },
  sectionLabel: {
    margin: "0 0 12px 4px",
    fontSize: 13,
    fontWeight: 700,
    color: "rgba(255, 255, 255, 0.55)",
    letterSpacing: "0.04em",
    textTransform: "uppercase",
  },
  card: {
    background: palette.white,
    borderRadius: 26,
    padding: "28px 26px 24px",
    boxShadow: "0 24px 60px -18px rgba(0, 0, 0, 0.45)",
  },
};

const formStyles: Record<string, CSSProperties> = {
  form: { display: "grid", gap: 16 },
  label: { display: "grid", gap: 7, fontSize: 14, fontWeight: 600, color: palette.ink },
  input: {
    borderColor: "#e2e8f0",
    background: "#fafcfe",
  },
  otpBlock: { display: "grid", gap: 12 },
  secondaryButton: {
    padding: "12px 16px",
    fontSize: 15,
    fontWeight: 700,
    borderRadius: 999,
    border: "1px solid #e2e8f0",
    cursor: "pointer",
    color: palette.ink,
    background: "#f8fafc",
  },
  info: {
    margin: 0,
    fontSize: 14,
    color: palette.muted,
  },
  success: {
    margin: 0,
    fontSize: 14,
    fontWeight: 600,
    color: "#16a34a",
  },
  checkboxLabel: {
    display: "flex",
    alignItems: "center",
    gap: 10,
    fontSize: 14,
    color: palette.muted,
  },
  error: {
    margin: 0,
    padding: "11px 13px",
    borderRadius: 12,
    background: "#fef2f2",
    color: "#dc2626",
    fontSize: 14,
    fontWeight: 500,
  },
  button: {
    marginTop: 4,
    padding: "14px 16px",
    fontSize: 16,
    fontWeight: 700,
    borderRadius: 999,
    border: "none",
    cursor: "pointer",
    color: palette.ink,
    background: palette.gold,
    boxShadow: "0 12px 28px -10px rgba(240, 180, 41, 0.55)",
  },
  footer: { marginTop: 22, textAlign: "center", fontSize: 14, color: palette.muted },
  link: { color: palette.ink, fontWeight: 700, textDecoration: "underline" },
};
