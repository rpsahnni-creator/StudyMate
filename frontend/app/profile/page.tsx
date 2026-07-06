"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { API_URL, fetchWithAuth } from "../../lib/auth";
import { useAuth } from "../../components/AuthProvider";
import { SubscriptionBadge } from "../../components/SubscriptionBadge";
import {
  getMySubscription,
  planDisplayName,
  scansLabel,
  type Entitlements,
} from "../../lib/billing";

interface MeResponse {
  id: number;
  name: string;
  email: string;
  role?: string;
  email_verified?: boolean;
}

export default function ProfilePage() {
  const { isLoggedIn, isLoading: authLoading, logout } = useAuth();
  const [me, setMe] = useState<MeResponse | null>(null);
  const [entitlements, setEntitlements] = useState<Entitlements | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (authLoading) return;
    if (!isLoggedIn) {
      setLoading(false);
      return;
    }
    async function load() {
      try {
        const [meRes, sub] = await Promise.all([
          fetchWithAuth(`${API_URL}/auth/me`).then(async (res) => {
            if (!res.ok) throw new Error("Failed to load profile");
            return (await res.json()) as MeResponse;
          }),
          getMySubscription(),
        ]);
        setMe(meRes);
        setEntitlements(sub);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load profile");
      } finally {
        setLoading(false);
      }
    }
    void load();
  }, [authLoading, isLoggedIn]);

  if (authLoading || loading) {
    return (
      <main style={styles.page}>
        <p style={styles.muted}>Loading profile…</p>
      </main>
    );
  }

  if (!isLoggedIn) {
    return (
      <main style={styles.page}>
        <p>Please <Link href="/auth/login?from=/profile">log in</Link> to view your profile.</p>
      </main>
    );
  }

  return (
    <main style={styles.page} className="animate-in">
      <div style={styles.header}>
        <div style={styles.identity}>
          <span style={styles.avatar}>
            {(me?.name ?? "?").trim().charAt(0).toUpperCase()}
          </span>
          <div>
            <h1 style={styles.title}>{me?.name ?? "Profile"}</h1>
            <p style={styles.email}>{me?.email ?? ""}</p>
          </div>
        </div>
        <SubscriptionBadge />
      </div>

      {error ? <p style={styles.error}>{error}</p> : null}

      <section style={styles.section}>
        <h2 style={styles.sectionTitle}>Account</h2>
        <dl style={styles.dl}>
          <dt>Name</dt>
          <dd>{me?.name ?? "—"}</dd>
          <dt>Email</dt>
          <dd>{me?.email ?? "—"}</dd>
        </dl>
        <Link href="/auth/login" style={styles.link}>
          Change password (re-login flow)
        </Link>
      </section>

      <section style={styles.section}>
        <h2 style={styles.sectionTitle}>Subscription</h2>
        {entitlements ? (
          <dl style={styles.dl}>
            <dt>Plan</dt>
            <dd>{planDisplayName(entitlements.plan)}</dd>
            <dt>Status</dt>
            <dd>{entitlements.has_active_sub ? "Active" : "Free tier"}</dd>
            <dt>Daily scan allowance</dt>
            <dd>{scansLabel(entitlements)}</dd>
            {entitlements.expires_at ? (
              <>
                <dt>Expires</dt>
                <dd>
                  {new Date(entitlements.expires_at).toLocaleDateString()}
                  {entitlements.days_remaining < 7
                    ? ` (${entitlements.days_remaining} days left)`
                    : ""}
                </dd>
              </>
            ) : null}
          </dl>
        ) : (
          <p style={styles.muted}>No subscription data</p>
        )}
        {entitlements?.plan !== "pro" ? (
          <Link href="/plans" style={styles.upgradeBtn}>
            Upgrade plan
          </Link>
        ) : null}
      </section>

      <button type="button" onClick={logout} className="btn-reset" style={styles.logoutBtn}>
        Log out
      </button>
    </main>
  );
}

const styles: Record<string, React.CSSProperties> = {
  page: {
    maxWidth: 660,
    margin: "0 auto",
    padding: "36px 20px 56px",
    display: "grid",
    gap: 22,
  },
  header: {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 12,
    flexWrap: "wrap",
  },
  identity: { display: "flex", alignItems: "center", gap: 14 },
  avatar: {
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    width: 54,
    height: 54,
    borderRadius: 16,
    background: "var(--brand-gradient)",
    color: "#fff",
    fontSize: 24,
    fontWeight: 800,
    boxShadow: "var(--shadow-brand)",
  },
  title: {
    margin: 0,
    fontSize: 26,
    fontWeight: 800,
    lineHeight: 1.15,
  },
  email: { margin: "2px 0 0", color: "var(--text-muted)", fontSize: 14 },
  section: {
    border: "1px solid var(--border)",
    borderRadius: "var(--r-lg)",
    padding: 22,
    background: "var(--surface)",
    boxShadow: "var(--shadow-sm)",
  },
  sectionTitle: {
    margin: "0 0 14px",
    fontSize: 18,
    fontWeight: 750,
  },
  dl: {
    display: "grid",
    gridTemplateColumns: "150px 1fr",
    gap: "10px 16px",
    margin: "0 0 14px",
    fontSize: 14.5,
  },
  link: {
    color: "var(--brand-600)",
    fontSize: 14,
    fontWeight: 600,
    textDecoration: "none",
  },
  upgradeBtn: {
    display: "inline-block",
    marginTop: 8,
    padding: "11px 18px",
    background: "var(--brand-gradient)",
    color: "#fff",
    borderRadius: "var(--r-md)",
    fontWeight: 700,
    textDecoration: "none",
    fontSize: 14,
    boxShadow: "var(--shadow-brand)",
  },
  logoutBtn: {
    justifySelf: "start",
    border: "1px solid var(--border-strong)",
    background: "var(--surface)",
    borderRadius: "var(--r-md)",
    padding: "11px 18px",
    cursor: "pointer",
    fontWeight: 600,
    color: "var(--text)",
  },
  muted: { color: "var(--text-muted)" },
  error: { color: "var(--danger)" },
};
