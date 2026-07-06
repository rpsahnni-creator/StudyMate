"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Crown, Sparkles, Zap } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import { useAuth } from "./AuthProvider";
import {
  getMySubscription,
  planDisplayName,
  type Entitlements,
} from "../lib/billing";

const PLAN_STYLES: Record<string, { bg: string; color: string; border: string; icon: LucideIcon }> = {
  free: { bg: "#f3f4f6", color: "#374151", border: "#d1d5db", icon: Zap },
  basic: { bg: "#eff6ff", color: "#1d4ed8", border: "#93c5fd", icon: Sparkles },
  pro: { bg: "#fffbeb", color: "#b45309", border: "#fcd34d", icon: Crown },
};

export function SubscriptionBadge() {
  const { isLoggedIn, isLoading: authLoading } = useAuth();
  const [entitlements, setEntitlements] = useState<Entitlements | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!isLoggedIn) {
      setEntitlements(null);
      return;
    }
    setLoading(true);
    void getMySubscription()
      .then(setEntitlements)
      .catch(() => setEntitlements(null))
      .finally(() => setLoading(false));
  }, [isLoggedIn]);

  if (authLoading || !isLoggedIn) return null;
  if (loading && !entitlements) {
    return <span style={styles.skeleton}>Plan…</span>;
  }

  const plan = entitlements?.plan ?? "free";
  const theme = PLAN_STYLES[plan] ?? PLAN_STYLES.free;
  const Icon = theme.icon;
  const label = planDisplayName(plan);
  const warnExpiry =
    entitlements?.has_active_sub &&
    entitlements.days_remaining >= 0 &&
    entitlements.days_remaining < 7;

  return (
    <Link href="/profile" style={{ textDecoration: "none" }}>
      <span
        style={{
          ...styles.badge,
          backgroundColor: theme.bg,
          color: theme.color,
          border: `1px solid ${theme.border}`,
        }}
        title={warnExpiry ? `${entitlements?.days_remaining} days left` : undefined}
      >
        <Icon size={14} strokeWidth={2.6} />
        {label}
        {warnExpiry ? ` · ${entitlements?.days_remaining}d` : ""}
      </span>
    </Link>
  );
}

const styles: Record<string, React.CSSProperties> = {
  badge: {
    display: "inline-flex",
    alignItems: "center",
    gap: 6,
    padding: "6px 12px",
    borderRadius: 999,
    fontSize: 14,
    fontWeight: 600,
  },
  skeleton: {
    fontSize: 14,
    color: "#9ca3af",
  },
};
