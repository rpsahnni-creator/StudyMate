import type { ResolvedFlags } from "../../shared/types/featureFlags";

let cachedFlags: ResolvedFlags | null = null;

export function getFeatureFlagCache(): ResolvedFlags | null {
  return cachedFlags;
}

export function setFeatureFlagCache(flags: ResolvedFlags): void {
  cachedFlags = flags;
}

export function clearFeatureFlagCache(): void {
  cachedFlags = null;
}
