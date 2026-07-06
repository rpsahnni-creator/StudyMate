import { API_URL, fetchWithAuth } from "./auth";

export interface Plan {
  id: string;
  name: string;
  price: number;
  currency: string;
  interval: string;
  features: string[];
  isPopular: boolean;
}

export interface Entitlements {
  has_active_sub: boolean;
  plan: string;
  scans_per_day: number;
  expires_at?: string;
  days_remaining: number;
}

export interface CheckoutPrefill {
  name?: string;
  email?: string;
}

export interface CheckoutResponse {
  orderId: string;
  amount: number;
  currency: string;
  keyId?: string;
  prefill?: CheckoutPrefill;
  provider: string;
}

export const FREE_PLAN: Plan = {
  id: "free",
  name: "Free",
  price: 0,
  currency: "INR",
  interval: "monthly",
  features: ["5 scans/day", "Basic quiz", "Community support"],
  isPopular: false,
};

export async function getPlans(): Promise<Plan[]> {
  const res = await fetch(`${API_URL}/plans`, { cache: "no-store" });
  if (!res.ok) {
    throw new Error("Failed to load plans");
  }
  return (await res.json()) as Plan[];
}

export async function getMySubscription(): Promise<Entitlements> {
  const res = await fetchWithAuth(`${API_URL}/users/me/subscription`);
  if (!res.ok) {
    throw new Error("Failed to load subscription");
  }
  return (await res.json()) as Entitlements;
}

export async function createCheckout(
  planId: string,
  provider = "razorpay"
): Promise<CheckoutResponse> {
  const res = await fetchWithAuth(`${API_URL}/billing/checkout`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ planId, provider }),
  });
  if (!res.ok) {
    let message = "Checkout failed";
    try {
      const data = (await res.json()) as { error?: string };
      message = data.error ?? message;
    } catch {
      // keep default
    }
    throw new Error(message);
  }
  return (await res.json()) as CheckoutResponse;
}

export function formatPlanPrice(plan: Plan): string {
  if (plan.price === 0) return "Free";
  return `₹${plan.price}/${plan.interval === "monthly" ? "mo" : plan.interval}`;
}

export function scansLabel(ent: Entitlements): string {
  if (ent.scans_per_day < 0) return "Unlimited scans/day";
  return `${ent.scans_per_day} scans/day`;
}

export function planDisplayName(plan: string): string {
  switch (plan) {
    case "pro":
      return "Pro";
    case "basic":
      return "Basic";
    default:
      return "Free";
  }
}
