"use client";

import { FormEvent, useState, type CSSProperties } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  API_URL,
  parseAuthError,
  persistSession,
  type TokenResponse,
} from "../../../lib/auth";

export default function RegisterPage() {
  const router = useRouter();

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");
  const [acceptTerms, setAcceptTerms] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);

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
          name,
          email,
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
        body: JSON.stringify({ email, password }),
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
    <main style={styles.main}>
      <div style={styles.card} className="animate-in">
        <Link href="/" style={styles.brand}>
          <span style={styles.logoMark}>S</span>
          <span style={styles.brandText}>StudyApp</span>
        </Link>
        <h1 style={styles.title}>Create your account</h1>
        <p style={styles.subtitle}>Start scanning chapters and generating quizzes.</p>

        <form onSubmit={handleSubmit} style={styles.form}>
          <label style={styles.label}>
            Name
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              autoComplete="name"
              minLength={2}
              placeholder="Your name"
            />
          </label>

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
              autoComplete="new-password"
              minLength={8}
              placeholder="At least 8 characters"
            />
          </label>

          <label style={styles.label}>
            Confirm password
            <input
              type="password"
              value={passwordConfirm}
              onChange={(e) => setPasswordConfirm(e.target.value)}
              required
              autoComplete="new-password"
              minLength={8}
              placeholder="Re-enter password"
            />
          </label>

          <label style={styles.checkboxLabel}>
            <input
              type="checkbox"
              checked={acceptTerms}
              onChange={(e) => setAcceptTerms(e.target.checked)}
            />
            <span>I accept the terms of service</span>
          </label>

          {error ? <p style={styles.error}>{error}</p> : null}

          <button type="submit" disabled={loading} style={styles.button}>
            {loading ? "Creating account…" : "Create account"}
          </button>
        </form>

        <p style={styles.footer}>
          Already have an account?{" "}
          <Link href="/auth/login" style={styles.link}>
            Sign in
          </Link>
        </p>
      </div>
    </main>
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
  checkboxLabel: {
    display: "flex",
    alignItems: "center",
    gap: 10,
    fontSize: 14,
    color: "var(--text-muted)",
  },
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
