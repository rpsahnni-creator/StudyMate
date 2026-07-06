"use client";

import { FormEvent, Suspense, useState, type CSSProperties } from "react";
import Link from "next/link";
import { Outfit } from "next/font/google";
import { useRouter, useSearchParams } from "next/navigation";
import { KijiLogo } from "../../components/KijiLogo";
import { parseAuthError, resetPassword } from "../../lib/auth";

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

function ResetPasswordForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const tokenFromUrl = searchParams.get("token") ?? "";

  const [token, setToken] = useState(tokenFromUrl);
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [done, setDone] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);

    if (!token.trim()) {
      setError("Reset link is invalid or expired. Request a new reset email.");
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
    <main style={styles.main}>
      <div style={styles.glowCyan} aria-hidden />
      <div style={styles.glowOrange} aria-hidden />
      <div style={styles.glowViolet} aria-hidden />

      <div style={styles.content} className="animate-in">
        <div style={styles.hero}>
          <Link href="/" style={styles.logoWrap}>
            <KijiLogo width={280} height={100} className="kiji-logo-hero" priority />
          </Link>
          <p style={styles.brandName}>Kiji Technology</p>
          <h1 className={titleFont.className} style={styles.heroTitle}>
            <span style={styles.heroStudy}>Study</span>
            <span style={styles.heroMate}>Mate</span>
          </h1>
        </div>

        <div style={styles.formSection}>
          <p style={styles.sectionLabel}>{done ? "Password updated" : "Reset password"}</p>
          <div style={styles.card}>
            {done ? (
              <>
                <p style={styles.subtitle}>You can now sign in with your new password.</p>
                <button
                  type="button"
                  className="btn-reset"
                  style={styles.button}
                  onClick={() => router.push("/auth/login")}
                >
                  Go to sign in
                </button>
              </>
            ) : (
              <form onSubmit={handleSubmit} style={styles.form}>
                {!tokenFromUrl ? (
                  <label style={styles.label}>
                    Reset code
                    <input
                      type="text"
                      value={token}
                      onChange={(e) => setToken(e.target.value)}
                      placeholder="Paste token from email link"
                      style={styles.input}
                    />
                  </label>
                ) : null}

                <label style={styles.label}>
                  New password
                  <input
                    type="password"
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    required
                    minLength={8}
                    autoComplete="new-password"
                    placeholder="At least 8 characters"
                    style={styles.input}
                  />
                </label>

                <label style={styles.label}>
                  Confirm new password
                  <input
                    type="password"
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    required
                    minLength={8}
                    autoComplete="new-password"
                    placeholder="Re-enter new password"
                    style={styles.input}
                  />
                </label>

                {error ? <p style={styles.error}>{error}</p> : null}

                <button type="submit" disabled={loading} className="btn-reset" style={styles.button}>
                  {loading ? "Updating…" : "Reset password"}
                </button>
              </form>
            )}

            <p style={styles.footer}>
              <Link href="/auth/login" style={styles.link}>
                Back to sign in
              </Link>
            </p>
          </div>
        </div>
      </div>
    </main>
  );
}

export default function ResetPasswordPage() {
  return (
    <Suspense fallback={<p style={{ padding: 24, textAlign: "center" }}>Loading…</p>}>
      <ResetPasswordForm />
    </Suspense>
  );
}

const styles: Record<string, CSSProperties> = {
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
  hero: { textAlign: "center", marginBottom: 28 },
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
    letterSpacing: "0.04em",
  },
  heroTitle: { margin: 0, fontSize: 40, lineHeight: 1.1, textAlign: "center" },
  heroStudy: { fontWeight: 300, color: palette.white, letterSpacing: "0.12em" },
  heroMate: {
    fontWeight: 800,
    fontStyle: "italic",
    color: palette.gold,
  },
  formSection: { width: "100%" },
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
  subtitle: { margin: "0 0 20px", color: palette.muted, fontSize: 15, lineHeight: 1.55 },
  form: { display: "grid", gap: 16 },
  label: { display: "grid", gap: 7, fontSize: 14, fontWeight: 600, color: palette.ink },
  input: { borderColor: "#e2e8f0", background: "#fafcfe" },
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
    width: "100%",
  },
  footer: { marginTop: 22, textAlign: "center", fontSize: 14, color: palette.muted },
  link: { color: palette.ink, fontWeight: 700, textDecoration: "underline" },
};
