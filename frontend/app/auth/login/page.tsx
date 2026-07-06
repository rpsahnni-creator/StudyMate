"use client";

import { FormEvent, Suspense, useState, type CSSProperties } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import {
  API_URL,
  getSafeRedirect,
  parseAuthError,
  persistSession,
  type TokenResponse,
} from "../../../lib/auth";

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
      <div style={styles.card} className="animate-in">
        <Link href="/" style={styles.brand}>
          <span style={styles.logoMark}>S</span>
          <span style={styles.brandText}>StudyApp</span>
        </Link>
        <h1 style={styles.title}>Welcome back</h1>
        <p style={styles.subtitle}>Sign in to access scans, quizzes, and your reports.</p>

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
            />
          </label>

          {error ? <p style={styles.error}>{error}</p> : null}

          <button type="submit" disabled={loading} style={styles.button}>
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
    </main>
  );
}

export default function LoginPage() {
  return (
    <Suspense fallback={<main style={styles.main}>Loading…</main>}>
      <LoginForm />
    </Suspense>
  );
}

const styles: Record<string, CSSProperties> = {
  main: {
    minHeight: "100vh",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    padding: 24,
    background:
      "radial-gradient(1000px 600px at 100% 0%, #eef2ff 0%, rgba(238,242,255,0) 55%), radial-gradient(900px 500px at 0% 100%, #faf5ff 0%, rgba(250,245,255,0) 55%), #f6f7fb",
  },
  card: {
    width: "100%",
    maxWidth: 430,
    background: "var(--surface)",
    borderRadius: "var(--r-xl)",
    padding: 36,
    boxShadow: "var(--shadow-lg)",
    border: "1px solid var(--border)",
  },
  brand: {
    display: "inline-flex",
    alignItems: "center",
    gap: 10,
    textDecoration: "none",
    marginBottom: 24,
  },
  logoMark: {
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    width: 36,
    height: 36,
    borderRadius: 11,
    background: "var(--brand-gradient)",
    color: "#fff",
    fontWeight: 800,
    fontSize: 19,
    boxShadow: "var(--shadow-brand)",
  },
  brandText: { fontWeight: 800, fontSize: 19, color: "var(--text)" },
  title: { margin: "0 0 6px", fontSize: 27, fontWeight: 800 },
  subtitle: { margin: "0 0 26px", color: "var(--text-muted)", fontSize: 15 },
  form: { display: "grid", gap: 16 },
  label: { display: "grid", gap: 7, fontSize: 14, fontWeight: 600 },
  error: {
    margin: 0,
    padding: "11px 13px",
    borderRadius: "var(--r-md)",
    background: "var(--danger-bg)",
    color: "var(--danger)",
    fontSize: 14,
    fontWeight: 500,
  },
  button: {
    marginTop: 4,
    padding: "13px 16px",
    fontSize: 16,
  },
  footer: { marginTop: 24, textAlign: "center", fontSize: 14, color: "var(--text-muted)" },
  link: { color: "var(--brand-600)", fontWeight: 700 },
};
