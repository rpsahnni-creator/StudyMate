"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import Script from "next/script";
import { useAuth } from "../../../components/AuthProvider";
import {
  createCheckout,
  formatPlanPrice,
  getPlans,
  type CheckoutResponse,
  type Plan,
} from "../../../lib/billing";

function CheckoutContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const planId = searchParams.get("planId") ?? "";
  const { isLoggedIn, isLoading: authLoading } = useAuth();

  const [plan, setPlan] = useState<Plan | null>(null);
  const [loadingPlan, setLoadingPlan] = useState(true);
  const [paying, setPaying] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sdkReady, setSdkReady] = useState(false);

  useEffect(() => {
    if (!authLoading && !isLoggedIn) {
      router.replace(`/auth/login?from=${encodeURIComponent(`/plans/checkout?planId=${planId}`)}`);
    }
  }, [authLoading, isLoggedIn, router, planId]);

  useEffect(() => {
    if (!planId) {
      setError("Missing plan. Choose a plan first.");
      setLoadingPlan(false);
      return;
    }
    void getPlans()
      .then((plans) => {
        const found = plans.find((p) => p.id === planId) ?? null;
        if (!found) {
          setError("Plan not found");
        }
        setPlan(found);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load plan"))
      .finally(() => setLoadingPlan(false));
  }, [planId]);

  async function openRazorpay(checkout: CheckoutResponse) {
    if (!window.Razorpay) {
      throw new Error("Razorpay SDK not loaded");
    }
    if (!checkout.keyId) {
      throw new Error("Missing Razorpay key");
    }

    const rzp = new window.Razorpay({
      key: checkout.keyId,
      order_id: checkout.orderId,
      amount: checkout.amount,
      currency: checkout.currency,
      name: "StudyApp",
      description: plan ? `${plan.name} subscription` : "Subscription",
      prefill: checkout.prefill,
      handler: () => {
        router.push(`/plans/success?orderId=${encodeURIComponent(checkout.orderId)}`);
      },
      modal: {
        ondismiss: () => setPaying(false),
      },
    });
    rzp.open();
  }

  async function handlePay() {
    if (!plan) return;
    setError(null);
    setPaying(true);
    try {
      const checkout = await createCheckout(plan.id, "razorpay");
      await openRazorpay(checkout);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Checkout failed");
      setPaying(false);
    }
  }

  if (authLoading || loadingPlan) {
    return <p style={styles.muted}>Loading checkout…</p>;
  }

  return (
    <>
      <Script
        src="https://checkout.razorpay.com/v1/checkout.js"
        strategy="afterInteractive"
        onLoad={() => setSdkReady(true)}
        onError={() => setError("Failed to load Razorpay SDK")}
      />

      <div style={styles.wrap} className="animate-in">
        <Link href="/plans" style={styles.back}>
          ← Back to plans
        </Link>
        <h1 style={styles.title}>Checkout</h1>

        {plan ? (
          <div style={styles.summary} className="card">
            <span style={styles.planName}>{plan.name}</span>
            <p style={styles.price}>{formatPlanPrice(plan)}</p>
            <ul style={styles.features}>
              {plan.features.map((f) => (
                <li key={f} style={styles.featureItem}>
                  <span style={styles.check}>✓</span> {f}
                </li>
              ))}
            </ul>
          </div>
        ) : null}

        {error ? <p style={styles.error}>{error}</p> : null}

        <button
          type="button"
          style={styles.payBtn}
          disabled={!plan || paying || !sdkReady}
          onClick={() => void handlePay()}
        >
          {paying ? "Opening Razorpay…" : !sdkReady ? "Loading payment…" : "Pay securely →"}
        </button>

        <p style={styles.note}>🔒 Payment opens in a secure Razorpay modal — you won&apos;t leave this page.</p>
      </div>
    </>
  );
}

export default function CheckoutPage() {
  return (
    <main style={styles.page}>
      <Suspense fallback={<p style={styles.muted}>Loading…</p>}>
        <CheckoutContent />
      </Suspense>
    </main>
  );
}

const styles: Record<string, React.CSSProperties> = {
  page: {
    maxWidth: 560,
    margin: "0 auto",
    padding: "32px 20px",
  },
  wrap: {
    display: "grid",
    gap: 16,
  },
  back: {
    color: "var(--brand-600)",
    textDecoration: "none",
    fontSize: 14,
    fontWeight: 600,
  },
  title: {
    margin: 0,
    fontSize: 30,
    fontWeight: 850,
  },
  summary: {
    padding: 24,
    display: "grid",
    gap: 4,
  },
  planName: {
    fontSize: 15,
    fontWeight: 700,
    color: "var(--brand-700)",
    textTransform: "uppercase",
    letterSpacing: "0.04em",
  },
  price: {
    margin: "2px 0 14px",
    fontSize: 34,
    fontWeight: 850,
  },
  features: {
    margin: 0,
    paddingLeft: 0,
    listStyle: "none",
    display: "grid",
    gap: 9,
    color: "var(--text)",
    fontSize: 14.5,
  },
  featureItem: { display: "flex", gap: 8, alignItems: "center" },
  check: { color: "var(--success)", fontWeight: 800 },
  payBtn: {
    border: "none",
    borderRadius: "var(--r-md)",
    padding: "15px 18px",
    background: "var(--brand-gradient)",
    color: "#fff",
    fontWeight: 700,
    fontSize: 16,
    cursor: "pointer",
    boxShadow: "var(--shadow-brand)",
  },
  note: {
    fontSize: 13,
    color: "var(--text-muted)",
    margin: 0,
    textAlign: "center",
  },
  muted: { color: "var(--text-muted)" },
  error: {
    color: "var(--danger)",
    margin: 0,
    background: "var(--danger-bg)",
    padding: "11px 13px",
    borderRadius: "var(--r-md)",
    fontWeight: 500,
  },
};
