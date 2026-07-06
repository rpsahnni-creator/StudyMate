"use client";

import { FormEvent, useEffect, useState } from "react";
import Link from "next/link";
import { API_URL, fetchWithAuth } from "../../lib/auth";
import {
  getMySubscription,
  planDisplayName,
  scansLabel,
  type Entitlements,
} from "../../lib/billing";

export default function ScanPage() {
  const [mode, setMode] = useState("chapter");
  const [board, setBoard] = useState("ncert");
  const [acceptedTerms, setAcceptedTerms] = useState(false);
  const [termsError, setTermsError] = useState(false);
  const [status, setStatus] = useState<string | null>(null);
  const [isError, setIsError] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [entitlements, setEntitlements] = useState<Entitlements | null>(null);
  const [loadingSub, setLoadingSub] = useState(true);

  useEffect(() => {
    void getMySubscription()
      .then(setEntitlements)
      .catch(() => setEntitlements(null))
      .finally(() => setLoadingSub(false));
  }, []);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    // Enforce the consent checkbox client-side too — the backend doesn't
    // reject accepted_terms=false today, so without this guard a user could
    // submit a scan job without ever agreeing to the copyright terms.
    if (!acceptedTerms) {
      setTermsError(true);
      setIsError(true);
      setStatus("Please confirm the terms above before scanning.");
      return;
    }
    setTermsError(false);

    setSubmitting(true);
    setIsError(false);
    setStatus("Creating scan job...");

    // Note: book_id/chapter_id are intentionally omitted. They're optional
    // on the backend (used only once real book/chapter catalog selection
    // ships) — sending made-up IDs here caused every scan job to fail with
    // a foreign-key violation because those rows don't exist.
    const payload = {
      mode,
      board,
      accepted_terms: acceptedTerms,
      page_no: 1,
    };

    try {
      const response = await fetchWithAuth(`${API_URL}/scan/jobs`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
      });

      const data = (await response.json()) as { error?: string; message?: string; job?: { id?: number } };
      if (!response.ok) {
        setIsError(true);
        if (response.status === 429) {
          setStatus("Daily scan limit reached. Upgrade your plan for more scans.");
          return;
        }
        setStatus(data.error ?? data.message ?? "Scan job could not be created");
        return;
      }

      setStatus(`Scan job created: ${data.job?.id ?? "unknown"}`);
    } catch {
      setIsError(true);
      setStatus("Network error — please check your connection and try again.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main style={styles.page} className="animate-in">
      <header style={styles.header}>
        <span style={styles.eyebrow}>📷 Scan</span>
        <h1 style={styles.title}>Scan Chapter / Questions</h1>
        <p style={styles.subtitle}>
          Turn legally-acquired study material into AI quizzes with a copyright-safe,
          transformative processing pipeline.
        </p>
      </header>

      {!loadingSub && entitlements ? (
        <div style={styles.planBar}>
          <div>
            <span style={styles.planBadge}>{planDisplayName(entitlements.plan)}</span>
            <span style={styles.planScans}>{scansLabel(entitlements)}</span>
          </div>
          {entitlements.plan === "free" ? (
            <Link href="/plans" style={styles.upgradeLink}>
              Upgrade →
            </Link>
          ) : null}
        </div>
      ) : null}

      <form onSubmit={handleSubmit} style={styles.form} className="card">
        <div style={styles.row}>
          <label style={styles.field}>
            <span style={styles.fieldLabel}>Mode</span>
            <select value={mode} onChange={(e) => setMode(e.target.value)}>
              <option value="chapter">Chapter</option>
              <option value="practice">Practice</option>
            </select>
          </label>

          <label style={styles.field}>
            <span style={styles.fieldLabel}>Board</span>
            <select value={board} onChange={(e) => setBoard(e.target.value)}>
              <option value="ncert">NCERT</option>
              <option value="cbse">CBSE</option>
              <option value="state_board">State Board</option>
            </select>
          </label>
        </div>

        <label style={{ ...styles.terms, ...(termsError ? styles.termsError : {}) }}>
          <input
            type="checkbox"
            checked={acceptedTerms}
            onChange={(e) => {
              setAcceptedTerms(e.target.checked);
              if (e.target.checked) setTermsError(false);
            }}
          />
          <span style={styles.termsText}>
            I confirm I am scanning legally acquired NCERT or state-board material for
            personal educational use, and I am not redistributing the source content
            commercially.
          </span>
        </label>

        <button type="submit" style={styles.submit} disabled={submitting}>
          {submitting ? "Creating…" : "Create Scan Job"}
        </button>
      </form>

      {status ? (
        <p style={{ ...styles.status, ...(isError ? styles.statusError : styles.statusOk) }}>
          {status}
        </p>
      ) : null}
    </main>
  );
}

const styles: Record<string, React.CSSProperties> = {
  page: { padding: "36px 20px 56px", maxWidth: 620, margin: "0 auto", display: "grid", gap: 20 },
  header: { display: "grid", gap: 10 },
  eyebrow: {
    justifySelf: "start",
    padding: "5px 12px",
    borderRadius: 999,
    background: "var(--brand-50)",
    color: "var(--brand-700)",
    fontSize: 13,
    fontWeight: 700,
    border: "1px solid var(--brand-100)",
  },
  title: { margin: 0, fontSize: 30, fontWeight: 800 },
  subtitle: { margin: 0, color: "var(--text-muted)", fontSize: 15, lineHeight: 1.6 },
  planBar: {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 12,
    padding: "12px 16px",
    borderRadius: "var(--r-lg)",
    background: "var(--brand-gradient-soft)",
    border: "1px solid var(--brand-100)",
  },
  planBadge: {
    display: "inline-block",
    padding: "3px 10px",
    borderRadius: 999,
    background: "#fff",
    color: "var(--brand-700)",
    fontSize: 12,
    fontWeight: 700,
    marginRight: 10,
    boxShadow: "var(--shadow-xs)",
  },
  planScans: { fontSize: 14, color: "var(--text)", fontWeight: 600 },
  upgradeLink: { color: "var(--brand-600)", fontWeight: 700, fontSize: 14 },
  form: { display: "grid", gap: 18, padding: 22 },
  row: { display: "grid", gridTemplateColumns: "1fr 1fr", gap: 14 },
  field: { display: "grid", gap: 7 },
  fieldLabel: { fontSize: 13, fontWeight: 700, color: "var(--text)" },
  terms: {
    display: "flex",
    gap: 12,
    alignItems: "flex-start",
    padding: 16,
    borderRadius: "var(--r-md)",
    background: "var(--surface-2)",
    border: "1px solid var(--border)",
  },
  termsError: {
    borderColor: "var(--danger)",
    background: "var(--danger-bg)",
  },
  termsText: { fontSize: 13.5, color: "var(--text-muted)", lineHeight: 1.55, fontWeight: 500 },
  submit: { padding: "13px 18px", fontSize: 16 },
  status: {
    margin: 0,
    padding: "12px 16px",
    borderRadius: "var(--r-md)",
    fontSize: 14,
    fontWeight: 600,
  },
  statusOk: { background: "var(--success-bg)", color: "var(--success)" },
  statusError: { background: "var(--danger-bg)", color: "var(--danger)" },
};
