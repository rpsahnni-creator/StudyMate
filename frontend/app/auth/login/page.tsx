"use client";

import { FormEvent, Suspense, useState, type CSSProperties } from "react";
import Link from "next/link";
import { Outfit } from "next/font/google";
import { useRouter, useSearchParams } from "next/navigation";
import { KijiLogo } from "../../../components/KijiLogo";
import {
  API_URL,
  getSafeRedirect,
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
  seaGreen: "#20B2AA",
  brandMuted: "#9EB4C8",
};

function LoginForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const redirectTo = getSafeRedirect(searchParams.get("from"), "/scan");

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setLoading(true);

    try {
      const res = await fetch(`${API_URL}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });

      if (!res.ok) {
        setError(await parseAuthError(res));
        return;
      }

      const data = (await res.json()) as TokenResponse;
      await persistSession(data);
      router.push(redirectTo);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main style={styles.main}>
      <div style={styles.glowCyan} aria-hidden />
      <div style={styles.glowOrange} aria-hidden />
      <div style={styles.glowViolet} aria-hidden />

      <div style={styles.content}>
        <div style={styles.hero} className="animate-in">
          <Link href="/" style={styles.logoWrap}>
            <KijiLogo width={280} height={100} className="kiji-logo-hero" priority />
          </Link>
          <p style={styles.brandName}>Kiji Technology</p>
          <h1 className={titleFont.className} style={styles.heroTitle}>
            <span style={styles.heroStudy}>Study</span>
            <span style={styles.heroMate}>Mate</span>
          </h1>
          <p style={styles.heroSubtitle}>Learn Smarter Every Day</p>
        </div>

        <div style={styles.formSection} className="animate-in">
          <p style={styles.sectionLabel}>Sign in to your account</p>

          <div style={styles.card}>
            <form onSubmit={handleSubmit} style={styles.form}>
              <label style={styles.label}>
                Email
                <input
                  type="email"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                  autoComplete="email"
                  placeholder="you@example.com"
                  style={styles.input}
                />
              </label>

              <label style={styles.label}>
                Password
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  autoComplete="current-password"
                  minLength={8}
                  placeholder="••••••••"
                  style={styles.input}
                />
              </label>

              {error ? <p style={styles.error}>{error}</p> : null}

              <button type="submit" disabled={loading} className="btn-reset" style={styles.button}>
                {loading ? "Signing in…" : "Sign in"}
              </button>
            </form>

            <p style={styles.footer}>
              No account?{" "}
              <Link href="/auth/register" style={styles.link}>
                Create one
              </Link>
            </p>
          </div>
        </div>
      </div>
    </main>
  );
}

export default function LoginPage() {
  return (
    <Suspense
      fallback={
        <main style={styles.main}>
          <p style={{ color: "rgba(255,255,255,0.7)", textAlign: "center" }}>Loading…</p>
        </main>
      }
    >
      <LoginForm />
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
    display: "flex",
    flexDirection: "column",
    alignItems: "center",
  },
  hero: {
    width: "100%",
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
    maxWidth: 360,
    marginLeft: "auto",
    marginRight: "auto",
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
  form: { display: "grid", gap: 16 },
  label: { display: "grid", gap: 7, fontSize: 14, fontWeight: 600, color: palette.ink },
  input: {
    borderColor: "#e2e8f0",
    background: "#fafcfe",
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
    transition: "transform 0.12s ease, box-shadow 0.2s ease",
  },
  footer: { marginTop: 22, textAlign: "center", fontSize: 14, color: palette.muted },
  link: { color: palette.ink, fontWeight: 700, textDecoration: "underline" },
};
