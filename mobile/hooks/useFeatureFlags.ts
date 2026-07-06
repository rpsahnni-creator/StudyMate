import { useEffect, useState } from "react";
import type { ResolvedFlags, FlagKey } from "../../shared/types/featureFlags";
import { DEFAULT_FLAGS } from "../../shared/types/featureFlags";
import { apiCall, getAccessToken } from "../lib/auth";
import { getFeatureFlagCache, setFeatureFlagCache } from "../lib/featureFlagCache";
import { useAuth } from "./useAuth";

export { clearFeatureFlagCache } from "../lib/featureFlagCache";

export function useFeatureFlags(): ResolvedFlags {
  const { token, isLoggedIn } = useAuth();
  const [flags, setFlags] = useState<ResolvedFlags>(getFeatureFlagCache() ?? DEFAULT_FLAGS);

  useEffect(() => {
    if (!isLoggedIn || !token) {
      setFlags(DEFAULT_FLAGS);
      return;
    }

    let cancelled = false;

    async function loadFlags() {
      const accessToken = await getAccessToken();
      if (!accessToken || cancelled) return;

      const cached = getFeatureFlagCache();
      if (cached) {
        setFlags(cached);
        return;
      }

      try {
        const res = await apiCall("/me/features");
        if (!res.ok || cancelled) {
          setFlags(DEFAULT_FLAGS);
          return;
        }
        const data = (await res.json()) as ResolvedFlags;
        setFeatureFlagCache(data);
        if (!cancelled) {
          setFlags(data);
        }
      } catch {
        if (!cancelled) {
          setFlags(DEFAULT_FLAGS);
        }
      }
    }

    void loadFlags();

    return () => {
      cancelled = true;
    };
  }, [token, isLoggedIn]);

  return flags;
}

export function useFeatureFlag(key: FlagKey): boolean {
  const flags = useFeatureFlags();
  return flags[key] ?? false;
}
