import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { ActivityIndicator, StyleSheet, View } from "react-native";
import { useRouter, useSegments } from "expo-router";
import { API_URL, clearTokens, getAccessToken, parseApiError, saveTokens, type TokenResponse } from "../lib/auth";
import { clearFeatureFlagCache } from "../lib/featureFlagCache";

interface AuthContextValue {
  token: string | null;
  isLoggedIn: boolean;
  isLoading: boolean;
  login: (email: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    let mounted = true;
    void getAccessToken().then((stored) => {
      if (mounted) {
        setToken(stored);
        setIsLoading(false);
      }
    });
    return () => {
      mounted = false;
    };
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const normalizedEmail = email.trim().toLowerCase();
    let res: Response;
    try {
      res = await fetch(`${API_URL}/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ email: normalizedEmail, password }),
      });
    } catch {
      throw new Error(
        `Server tak nahi pahunch paaye (${API_URL}). PC par backend chal raha hai? Phone aur PC same Wi‑Fi par hain?`
      );
    }

    if (!res.ok) {
      const message = await parseApiError(res);
      if (res.status === 401) {
        throw new Error(`${message} — email/password check karo (demo: demo@123.com / Demo@123)`);
      }
      throw new Error(message);
    }

    const data = (await res.json()) as TokenResponse;
    if (!data.access_token) {
      throw new Error("Login response invalid — backend se token nahi mila");
    }
    await saveTokens(data.access_token, data.refresh_token);
    clearFeatureFlagCache();
    setToken(data.access_token);
  }, []);

  const logout = useCallback(async () => {
    await clearTokens();
    clearFeatureFlagCache();
    setToken(null);
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      token,
      isLoggedIn: Boolean(token),
      isLoading,
      login,
      logout,
    }),
    [token, isLoading, login, logout]
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

export function AuthGate({ children }: { children: ReactNode }) {
  const { isLoggedIn, isLoading } = useAuth();
  const segments = useSegments();
  const router = useRouter();

  useEffect(() => {
    if (isLoading) return;

    const inAuthGroup = segments[0] === "auth";

    if (!isLoggedIn && !inAuthGroup) {
      router.replace("/auth/login");
      return;
    }

    if (isLoggedIn && inAuthGroup) {
      router.replace("/(tabs)/scan");
    }
  }, [isLoggedIn, isLoading, segments, router]);

  if (isLoading) {
    return (
      <View style={styles.loading}>
        <ActivityIndicator size="large" color="#111" />
      </View>
    );
  }

  return <>{children}</>;
}

const styles = StyleSheet.create({
  loading: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: "#fff",
  },
});
