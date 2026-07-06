"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import {
  getMySubscription,
  planDisplayName,
  scansLabel,
  type Entitlements,
} from "../../../lib/billing";

const POLL_INTERVAL_MS = 1000;
const POLL_TIMEOUT_MS = 10000;

function SuccessContent() {
  const searchParams = useSearchParams();
  const orderId = searchParams.get("orderId");
  const [entitlements, setEntitlements] = useState<Entitlements | null>(null);
  const [polling, setPolling] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    const started = Date.now();

    async function poll() {
      while (!cancelled && Date.now() - started < POLL_TIMEOUT_MS) {
        try {
          const sub = await getMySubscription();
          if (sub.has_active_sub) {
            setEntitlements(sub);
            setPolling(false);
            return;
          }
        } catch (err) {
          if (!cancelled) {
            setError(err instanceof Error ? err.message : "Failed to confirm subscription");
          }
          break;
        }
        await new Promise((r) => setTimeout(r, POLL_INTERVAL_MS));
      }
      if (!cancelled) {
        setPolling(false);
      }
    }

    void poll();
    return () => {
      cancelled = true;
    };
  }, []);

  const active = entitlements?.has_active_sub;

  return (
    <div style={styles.wrap} className="animate-in">
      <div style={styles.icon}>✓</div>
      <h1 style={styles.title}>Payment successful!</h1>
      <p style={styles.subtitle}>
        {polling
          ? "Confirming your subscription…"
          : active
            ? "Your subscription is now active."
            : "Payment received. Subscription may take a moment to activate."}
      </p>

      {orderId ? <p style={styles.order}>Order: {orderId}</p> : null}

      {entitlements ? (
        <div style={styles.card} className="card">
          <p style={styles.row}>
            <span>Plan</span>
            <strong>{planDisplayName(entitlements.plan)}</strong>
          </p>
          <p style={styles.row}>
            <span>Daily scans</span>
            <strong>{scansLabel(entitlements)}</strong>
          </p>
          {entitlements.expires_at ? (
            <p style={styles.row}>
              <span>Renews / expires</span>
              <strong>{new Date(entitlements.expires_at).toLocaleDateString()}</strong>
            </p>
          ) : null}
        </div>
      ) : null}

      {error ? <p style={styles.error}>{error}</p> : null}

      <Link href="/scan" style={styles.cta}>
        Start studying
      </Link>
      <Link href="/profile" style={styles.secondary}>
        View profile
      </Link>
    </div>
  );
}

export default function SuccessPage() {
  return (
    <main style={styles.page}>
      <Suspense fallback={<p style={styles.muted}>Loading…</p>}>
        <SuccessContent />
      </Suspense>
    </main>
  );
}

const styles: Record<string, React.CSSProperties> = {
  page: {
    maxWidth: 520,
    margin: "0 auto",
    padding: "48px 20px",
    textAlign: "center",
  },
  wrap: {
    display: "grid",
    gap: 16,
    justifyItems: "center",
  },
  icon: {
    width: 72,
    height: 72,
    borderRadius: "50%",
    background: "linear-gradient(135deg,#22c55e,#16a34a)",
    color: "#fff",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    fontSize: 36,
    fontWeight: 800,
    boxShadow: "0 12px 30px -10px rgba(22,163,74,0.6)",
  },
  title: {
    margin: 0,
    fontSize: 30,
    fontWeight: 850,
    color: "var(--text)",
  },
  subtitle: {
    margin: 0,
    color: "var(--text-muted)",
    maxWidth: 400,
    fontSize: 15,
  },
  order: {
    margin: 0,
    fontSize: 13,
    color: "var(--text-subtle)",
  },
  card: {
    width: "100%",
    padding: 18,
    textAlign: "left",
    display: "grid",
    gap: 2,
  },
  row: {
    margin: "8px 0",
    display: "flex",
    justifyContent: "space-between",
    gap: 12,
    fontSize: 14.5,
    color: "var(--text)",
  },
  cta: {
    display: "inline-block",
    marginTop: 8,
    padding: "13px 26px",
    background: "var(--brand-gradient)",
    color: "#fff",
    borderRadius: "var(--r-md)",
    fontWeight: 700,
    textDecoration: "none",
    boxShadow: "var(--shadow-brand)",
  },
  secondary: {
    color: "var(--brand-600)",
    fontSize: 14,
    textDecoration: "none",
    fontWeight: 600,
  },
  muted: { color: "var(--text-muted)" },
  error: { color: "var(--danger)" },
};
