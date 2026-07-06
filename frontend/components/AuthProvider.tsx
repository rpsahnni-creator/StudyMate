"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { useRouter } from "next/navigation";
import {
  API_URL,
  clearTokens,
  fetchWithAuth,
  getStoredUser,
  getTokenExpiresAt,
  isLoggedIn,
  parseAuthError,
  persistSession,
  refreshAccessToken,
  type TokenResponse,
  type UserInfo,
} from "../lib/auth";

interface AuthContextValue {
  user: UserInfo | null;
  isLoggedIn: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const router = useRouter();
  const [user, setUser] = useState<UserInfo | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const loadUser = useCallback(async () => {
    if (!isLoggedIn()) {
      // The access token lives in memory only, so a fresh page load always
      // starts with none. Before treating the user as logged out, try to
      // silently redeem the httpOnly refresh-token cookie (if any) for a
      // new access token — this is what keeps sessions alive across reloads
      // without ever persisting the access token itself.
      const refreshed = await refreshAccessToken();
      if (!refreshed) {
        setUser(null);
        return;
      }
    }

    const cached = getStoredUser();
    if (cached) {
      setUser(cached);
    }

    try {
      const res = await fetchWithAuth(`${API_URL}/auth/me`);
      if (res.ok) {
        const me = (await res.json()) as UserInfo;
        setUser(me);
      } else if (res.status === 401) {
        setUser(null);
      }
    } catch {
      // Keep cached user on network errors
    }
  }, []);

  useEffect(() => {
    void loadUser().finally(() => setIsLoading(false));
  }, [loadUser]);

  useEffect(() => {
    if (!user && isLoggedIn()) {
      const cached = getStoredUser();
      if (cached) setUser(cached);
    }
  }, [user]);

  useEffect(() => {
    if (!isLoggedIn()) return;

    const expiresAt = getTokenExpiresAt();
    if (!expiresAt) return;

    const refreshAt = expiresAt - 5 * 60 * 1000;
    const delay = refreshAt - Date.now();

    const runRefresh = () => {
      void refreshAccessToken().then((token) => {
        if (token) {
          void loadUser();
        } else {
          setUser(null);
        }
      });
    };

    if (delay <= 0) {
      runRefresh();
      return;
    }

    const timer = window.setTimeout(runRefresh, delay);
    return () => window.clearTimeout(timer);
  }, [user, loadUser]);

  const login = useCallback(
    async (email: string, password: string) => {
      const res = await fetch(`${API_URL}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email, password }),
      });

      if (!res.ok) {
        throw new Error(await parseAuthError(res));
      }

      const data = (await res.json()) as TokenResponse;
      await persistSession(data);
      setUser(data.user);
      router.push("/scan");
    },
    [router]
  );

  const logout = useCallback(() => {
    clearTokens();
    setUser(null);
    router.push("/auth/login");
  }, [router]);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      isLoggedIn: user != null || isLoggedIn(),
      isLoading,
      login,
      logout,
    }),
    [user, isLoading, login, logout]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return ctx;
}
