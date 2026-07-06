"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "../../components/AuthProvider";
import {
  FREE_PLAN,
  formatPlanPrice,
  getMySubscription,
  getPlans,
  planDisplayName,
  type Entitlements,
  type Plan,
} from "../../lib/billing";

export default function PlansPage() {
  const router = useRouter();
  const { isLoggedIn } = useAuth();
  const [plans, setPlans] = useState<Plan[]>([]);
  const [entitlements, setEntitlements] = useState<Entitlements | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function load() {
      try {
        const apiPlans = await getPlans();
        setPlans(apiPlans);
        if (isLoggedIn) {
          const sub = await getMySubscription();
          setEntitlements(sub);
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : "Failed to load plans");
      } finally {
        setLoading(false);
      }
    }
    void load();
  }, [isLoggedIn]);

  function isCurrentPlan(planId: string): boolean {
    if (!entitlements) return planId === "free" && !isLoggedIn;
    if (planId === "free") return entitlements.plan === "free";
    if (planId === "plan_basic_monthly") return entitlements.plan === "basic";
    if (planId === "plan_pro_monthly") return entitlements.plan === "pro";
    return false;
  }

  function handleSubscribe(planId: string) {
    if (!isLoggedIn) {
      router.push(`/auth/login?from=${encodeURIComponent(`/plans/checkout?planId=${planId}`)}`);
      return;
    }
    router.push(`/plans/checkout?planId=${encodeURIComponent(planId)}`);
  }

  const allPlans = [FREE_PLAN, ...plans];

  return (
    <main style={styles.page} className="animate-in">
      <div style={styles.hero}>
        <span style={styles.eyebrow}>Pricing</span>
        <h1 style={styles.title}>Choose your plan</h1>
        <p style={styles.subtitle}>
          Unlock more daily scans and AI-powered quizzes. Upgrade anytime — subscriptions extend automatically.
        </p>
        {isLoggedIn && entitlements ? (
          <p style={styles.current}>
            Current plan: <strong>{planDisplayName(entitlements.plan)}</strong>
          </p>
        ) : null}
      </div>

      {loading ? <p style={styles.muted}>Loading plans…</p> : null}
      {error ? <p style={styles.error}>{error}</p> : null}

      <div style={styles.grid}>
        {allPlans.map((plan) => {
          const current = isCurrentPlan(plan.id);
          const isFree = plan.id === "free";
          return (
            <article
              key={plan.id}
              style={{
                ...styles.card,
                ...(plan.isPopular ? styles.cardPopular : {}),
                ...(current ? styles.cardCurrent : {}),
              }}
            >
              {plan.isPopular ? <span style={styles.popularBadge}>Most popular</span> : null}
              {current ? <span style={styles.currentBadge}>Your plan</span> : null}
              <h2 style={styles.planName}>{plan.name}</h2>
              <p style={styles.price}>
                {isFree ? "₹0 / forever" : formatPlanPrice(plan)}
              </p>
              <ul style={styles.featureList}>
                {plan.features.map((feature) => (
                  <li key={feature} style={styles.featureItem}>
                    <span style={styles.check}>✓</span>
                    {feature}
                  </li>
                ))}
              </ul>
              {isFree ? (
                <Link href="/scan" style={styles.secondaryBtn}>
                  Get started
                </Link>
              ) : current ? (
                <button type="button" className="btn-reset" disabled style={styles.disabledBtn}>
                  Current plan
                </button>
              ) : (
                <button
                  type="button"
                  className="btn-reset"
                  style={plan.isPopular ? styles.primaryBtn : styles.secondaryBtn}
                  onClick={() => handleSubscribe(plan.id)}
                >
                  Subscribe
                </button>
              )}
            </article>
          );
        })}
      </div>
    </main>
  );
}

const styles: Record<string, React.CSSProperties> = {
  page: {
    maxWidth: 1100,
    margin: "0 auto",
    padding: "32px 20px 48px",
  },
  hero: {
    textAlign: "center",
    marginBottom: 36,
    display: "grid",
    gap: 10,
    justifyItems: "center",
  },
  eyebrow: {
    padding: "5px 12px",
    borderRadius: 999,
    background: "rgba(240, 180, 41, 0.12)",
    color: "var(--gold)",
    fontSize: 13,
    fontWeight: 600,
    letterSpacing: "-0.02em",
    border: "1px solid rgba(240, 180, 41, 0.28)",
  },
  title: {
    fontSize: 36,
    fontWeight: 850,
    margin: 0,
    color: "var(--text)",
  },
  subtitle: {
    color: "var(--text-muted)",
    margin: 0,
    maxWidth: 560,
    marginInline: "auto",
    fontSize: 16,
  },
  current: {
    marginTop: 4,
    color: "var(--text-muted)",
  },
  muted: { textAlign: "center", color: "var(--text-muted)" },
  error: { textAlign: "center", color: "var(--danger)" },
  grid: {
    display: "grid",
    gridTemplateColumns: "repeat(auto-fit, minmax(280px, 1fr))",
    gap: 20,
    alignItems: "stretch",
  },
  card: {
    position: "relative",
    border: "1px solid var(--border)",
    borderRadius: "var(--r-xl)",
    padding: 26,
    background: "rgba(255, 255, 255, 0.06)",
    display: "flex",
    flexDirection: "column",
    gap: 12,
    height: "100%",
    minHeight: 400,
    boxShadow: "var(--shadow-sm)",
    transition: "border-color 0.18s ease, box-shadow 0.2s ease",
  },
  cardPopular: {
    borderColor: "rgba(240, 180, 41, 0.45)",
    boxShadow: "var(--shadow-lg), 0 0 0 1px rgba(240, 180, 41, 0.15)",
  },
  cardCurrent: {
    outline: "2px solid var(--brand-500)",
    outlineOffset: 2,
  },
  popularBadge: {
    position: "absolute",
    top: 16,
    right: 16,
    background: "var(--brand-gradient)",
    color: "#fff",
    fontSize: 11,
    fontWeight: 800,
    padding: "5px 10px",
    borderRadius: 999,
    textTransform: "uppercase",
    letterSpacing: "0.04em",
    boxShadow: "var(--shadow-brand)",
  },
  currentBadge: {
    alignSelf: "flex-start",
    background: "rgba(240, 180, 41, 0.12)",
    color: "var(--gold)",
    fontSize: 11,
    fontWeight: 600,
    letterSpacing: "-0.01em",
    padding: "4px 10px",
    borderRadius: 999,
    border: "1px solid rgba(240, 180, 41, 0.28)",
  },
  planName: {
    margin: 0,
    fontSize: 22,
    fontWeight: 600,
    letterSpacing: "-0.02em",
    color: "var(--text)",
  },
  price: {
    margin: 0,
    fontSize: 32,
    fontWeight: 850,
    color: "var(--text)",
  },
  featureList: {
    listStyle: "none",
    padding: 0,
    margin: "8px 0 0",
    flex: 1,
    display: "grid",
    gap: 8,
  },
  featureItem: {
    display: "flex",
    gap: 8,
    alignItems: "flex-start",
    color: "var(--text-muted)",
    fontSize: 14,
    fontWeight: 400,
    letterSpacing: "-0.01em",
    lineHeight: 1.45,
  },
  check: {
    color: "var(--success)",
    fontWeight: 800,
  },
  primaryBtn: {
    marginTop: "auto",
    border: "none",
    borderRadius: "var(--r-md)",
    padding: "13px 16px",
    background: "var(--brand-gradient)",
    color: "#fff",
    fontWeight: 600,
    letterSpacing: "-0.02em",
    cursor: "pointer",
    textAlign: "center",
    textDecoration: "none",
    boxShadow: "var(--shadow-brand)",
  },
  secondaryBtn: {
    marginTop: "auto",
    border: "1px solid var(--border-strong)",
    borderRadius: "var(--r-md)",
    padding: "13px 16px",
    background: "rgba(255, 255, 255, 0.06)",
    color: "var(--text)",
    fontWeight: 500,
    letterSpacing: "-0.02em",
    cursor: "pointer",
    textAlign: "center",
    textDecoration: "none",
    display: "block",
  },
  disabledBtn: {
    marginTop: "auto",
    border: "1px solid var(--border)",
    borderRadius: "var(--r-md)",
    padding: "13px 16px",
    background: "var(--surface-2)",
    color: "var(--text-subtle)",
    fontWeight: 650,
    cursor: "not-allowed",
  },
};
