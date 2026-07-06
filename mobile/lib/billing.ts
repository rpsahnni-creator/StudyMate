import { apiCall } from "./auth";

export interface Entitlements {
  has_active_sub: boolean;
  plan: string;
  scans_per_day: number;
  expires_at?: string;
  days_remaining: number;
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

export function scansLabel(entitlements: Entitlements): string {
  if (entitlements.scans_per_day < 0) return "Unlimited scans/day";
  return `${entitlements.scans_per_day} scans/day`;
}

export async function getMySubscription(): Promise<Entitlements | null> {
  try {
    const res = await apiCall("/users/me/subscription");
    if (!res.ok) return null;
    return (await res.json()) as Entitlements;
  } catch {
    return null;
  }
}
