"use client";

import { createContext, useContext, useEffect, useState, ReactNode } from "react";
import type { ResolvedFlags, FlagKey } from "../types/featureFlags";
import { DEFAULT_FLAGS } from "../types/featureFlags";
import { API_URL, fetchWithAuth, getToken } from "./auth";
import { useAuth } from "../components/AuthProvider";

const FeatureFlagsContext = createContext<ResolvedFlags>(DEFAULT_FLAGS);

export function FeatureFlagsProvider({ children }: { children: ReactNode }) {
  const [flags, setFlags] = useState<ResolvedFlags>(DEFAULT_FLAGS);
  const { user, isLoading } = useAuth();

  useEffect(() => {
    // Wait for AuthProvider to finish resolving the session (including its
    // silent refresh-on-boot) before deciding whether to fetch flags. This
    // also means flags are automatically re-fetched whenever `user` changes
    // — i.e. right after login, right after logout, and after the boot-time
    // session check resolves — instead of only once on first mount.
    if (isLoading) return;

    if (!user || !getToken()) {
      setFlags(DEFAULT_FLAGS);
      return;
    }

    let cancelled = false;
    fetchWithAuth(`${API_URL}/me/features`)
      .then((res) => (res.ok ? res.json() : DEFAULT_FLAGS))
      .then((data: ResolvedFlags) => {
        if (!cancelled) setFlags(data);
      })
      .catch(() => {
        if (!cancelled) setFlags(DEFAULT_FLAGS);
      });

    return () => {
      cancelled = true;
    };
  }, [user, isLoading]);

  return (
    <FeatureFlagsContext.Provider value={flags}>
      {children}
    </FeatureFlagsContext.Provider>
  );
}

export function useFeatureFlag(key: FlagKey): boolean {
  const flags = useContext(FeatureFlagsContext);
  return flags[key] ?? false;
}
