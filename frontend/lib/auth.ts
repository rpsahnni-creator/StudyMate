const USER_KEY = "studyapp_user";
const REFRESH_COOKIE = "studyapp_refresh_token";

export const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export interface UserInfo {
  id: number;
  name: string;
  email: string;
  role: string;
}

export interface TokenResponse {
  access_token: string;
  refresh_token?: string;
  expires_in: number;
  token_type: string;
  user: UserInfo;
}

// --- Access token storage ---
//
// The access token is kept in memory only (module-level variables), never in
// localStorage/sessionStorage. This limits the damage an XSS payload can do:
// it can only read the token from a live JS context, not exfiltrate it from
// persisted storage after the page closes, and a reload always re-derives a
// fresh token from the httpOnly refresh-token cookie (see refreshAccessToken).
// The cookie itself is inaccessible to JavaScript, so it cannot be read by an
// injected script either.
let memoryToken: string | null = null;
let memoryTokenExpiresAt: number | null = null;
let refreshPromise: Promise<string | null> | null = null;

export function getToken(): string | null {
  return memoryToken;
}

export function getTokenExpiresAt(): number | null {
  return memoryTokenExpiresAt;
}

// User profile info (id/name/email/role) is not sensitive session material,
// so we cache it in sessionStorage purely to avoid a UI flash on reload.
// sessionStorage clears when the tab closes, unlike localStorage.
export function getStoredUser(): UserInfo | null {
  if (typeof window === "undefined") return null;
  const raw = sessionStorage.getItem(USER_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as UserInfo;
  } catch {
    return null;
  }
}

export function setToken(token: string, expiresInSeconds?: number): void {
  memoryToken = token;
  memoryTokenExpiresAt = expiresInSeconds != null ? Date.now() + expiresInSeconds * 1000 : null;
}

export function setStoredUser(user: UserInfo): void {
  if (typeof window === "undefined") return;
  sessionStorage.setItem(USER_KEY, JSON.stringify(user));
}

export function clearTokens(): void {
  memoryToken = null;
  memoryTokenExpiresAt = null;
  if (typeof window !== "undefined") {
    sessionStorage.removeItem(USER_KEY);
  }
  void fetch("/api/auth/set-cookie", { method: "DELETE" }).catch(() => {});
}

export function isLoggedIn(): boolean {
  if (!memoryToken) return false;
  if (memoryTokenExpiresAt != null && Date.now() >= memoryTokenExpiresAt) {
    return false;
  }
  return true;
}

export async function persistSession(data: TokenResponse): Promise<void> {
  setToken(data.access_token, data.expires_in);
  setStoredUser(data.user);

  if (data.refresh_token) {
    const res = await fetch("/api/auth/set-cookie", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: data.refresh_token }),
      credentials: "include",
    });
    if (!res.ok) {
      throw new Error("Failed to store refresh token");
    }
  }
}

// Redeems the httpOnly refresh-token cookie for a fresh access token. Safe to
// call speculatively (e.g. on app boot) since it simply fails closed (returns
// null) when no valid session cookie is present.
export async function refreshAccessToken(): Promise<string | null> {
  const res = await fetch("/api/auth/refresh", {
    method: "POST",
    credentials: "include",
  });

  if (!res.ok) {
    clearTokens();
    return null;
  }

  const data = (await res.json()) as TokenResponse;
  setToken(data.access_token, data.expires_in);
  if (data.user) {
    setStoredUser(data.user);
  }
  if (data.refresh_token) {
    await fetch("/api/auth/set-cookie", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: data.refresh_token }),
      credentials: "include",
    });
  }
  return data.access_token;
}

export async function fetchWithAuth(url: string, options: RequestInit = {}): Promise<Response> {
  const headers = new Headers(options.headers);
  const token = getToken();
  if (token) {
    headers.set("Authorization", `Bearer ${token}`);
  }

  let response = await fetch(url, { ...options, headers });

  if (response.status === 401) {
    if (!refreshPromise) {
      refreshPromise = refreshAccessToken().finally(() => {
        refreshPromise = null;
      });
    }
    const newToken = await refreshPromise;
    if (newToken) {
      headers.set("Authorization", `Bearer ${newToken}`);
      response = await fetch(url, { ...options, headers });
    }
  }

  return response;
}

export function getRefreshCookieName(): string {
  return REFRESH_COOKIE;
}

export async function parseAuthError(res: Response): Promise<string> {
  try {
    const data = (await res.json()) as { message?: string; error?: string };
    return data.message ?? data.error ?? `Request failed (${res.status})`;
  } catch {
    return `Request failed (${res.status})`;
  }
}

// --- Safe internal redirect helper ---
//
// Prevents open-redirect attacks via ?from=<url> style query params. Only
// same-document, absolute-path targets ("/foo/bar") are allowed; anything
// that looks like a protocol-relative URL ("//evil.com"), a full URL
// ("https://evil.com" or "javascript:..."), or a backslash trick
// ("/\evil.com", which some browsers normalize to "//evil.com") is rejected
// in favor of the provided fallback.
export function getSafeRedirect(target: string | null | undefined, fallback = "/"): string {
  if (!target) return fallback;
  if (!target.startsWith("/")) return fallback;
  if (target.startsWith("//")) return fallback;
  if (target.startsWith("/\\")) return fallback;
  if (target.includes("://")) return fallback;
  return target;
}
