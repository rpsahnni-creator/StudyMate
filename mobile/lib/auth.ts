import * as SecureStore from "expo-secure-store";
import AsyncStorage from "@react-native-async-storage/async-storage";
import { API_URL } from "./config";

export { API_URL };

export const TOKEN_KEY = "studyapp_access_token";
export const REFRESH_KEY = "studyapp_refresh_token";
export const USER_KEY = "studyapp_user";

export interface AuthUser {
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
  user: {
    id: number;
    name: string;
    email: string;
    role: string;
  };
}

let refreshPromise: Promise<string | null> | null = null;
let cachedAccessToken: string | null | undefined;
let secureStoreAvailable: boolean | null = null;

async function isSecureStoreAvailable(): Promise<boolean> {
  if (secureStoreAvailable !== null) {
    return secureStoreAvailable;
  }
  try {
    secureStoreAvailable = await SecureStore.isAvailableAsync();
  } catch {
    secureStoreAvailable = false;
  }
  return secureStoreAvailable;
}

async function storageGet(key: string): Promise<string | null> {
  if (await isSecureStoreAvailable()) {
    try {
      const value = await SecureStore.getItemAsync(key);
      if (value != null) return value;
    } catch {
      // fall through to AsyncStorage
    }
  }
  return AsyncStorage.getItem(key);
}

async function storageSet(key: string, value: string): Promise<void> {
  if (await isSecureStoreAvailable()) {
    try {
      await SecureStore.setItemAsync(key, value);
      return;
    } catch {
      // fall through to AsyncStorage
    }
  }
  await AsyncStorage.setItem(key, value);
}

async function storageDelete(key: string): Promise<void> {
  if (await isSecureStoreAvailable()) {
    try {
      await SecureStore.deleteItemAsync(key);
    } catch {
      // continue clearing AsyncStorage fallback
    }
  }
  await AsyncStorage.removeItem(key);
}

export async function saveTokens(access: string, refresh?: string): Promise<void> {
  cachedAccessToken = access;
  await storageSet(TOKEN_KEY, access);
  if (refresh) {
    await storageSet(REFRESH_KEY, refresh);
  }
}

export async function saveUser(user: AuthUser): Promise<void> {
  await storageSet(USER_KEY, JSON.stringify(user));
}

export async function getUser(): Promise<AuthUser | null> {
  const raw = await storageGet(USER_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as AuthUser;
  } catch {
    return null;
  }
}

export async function fetchCurrentUser(): Promise<AuthUser | null> {
  try {
    const res = await apiCall("/auth/me");
    if (!res.ok) return null;
    const data = (await res.json()) as AuthUser;
    if (!data?.name) return null;
    await saveUser(data);
    return data;
  } catch {
    return null;
  }
}

export async function getAccessToken(): Promise<string | null> {
  if (cachedAccessToken !== undefined) {
    return cachedAccessToken;
  }
  const token = await storageGet(TOKEN_KEY);
  cachedAccessToken = token;
  return token;
}

export async function getRefreshToken(): Promise<string | null> {
  return storageGet(REFRESH_KEY);
}

export async function clearTokens(): Promise<void> {
  cachedAccessToken = null;
  await storageDelete(TOKEN_KEY);
  await storageDelete(REFRESH_KEY);
  await storageDelete(USER_KEY);
}

export async function refreshAccessToken(): Promise<string | null> {
  const refreshToken = await getRefreshToken();
  if (!refreshToken) {
    await clearTokens();
    return null;
  }

  const res = await fetch(`${API_URL}/auth/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });

  if (!res.ok) {
    await clearTokens();
    return null;
  }

  const data = (await res.json()) as TokenResponse;
  await saveTokens(data.access_token, data.refresh_token ?? refreshToken);
  return data.access_token;
}

export async function requestPasswordReset(email: string): Promise<void> {
  const res = await fetch(`${API_URL}/auth/forgot-password`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email }),
  });
  if (!res.ok) {
    throw new Error(await parseApiError(res));
  }
}

export async function resetPassword(
  token: string,
  newPassword: string,
  newPasswordConfirm: string
): Promise<void> {
  const res = await fetch(`${API_URL}/auth/reset-password`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      token,
      new_password: newPassword,
      new_password_confirm: newPasswordConfirm,
    }),
  });
  if (!res.ok) {
    throw new Error(await parseApiError(res));
  }
}

export function formatApiErrorBody(data: {
  message?: string;
  error?: string;
  details?: Record<string, string> | string;
}): string {
  let msg = data.message ?? data.error ?? "Request failed";
  if (data.details && typeof data.details === "object" && !Array.isArray(data.details)) {
    const parts = Object.entries(data.details).map(([key, value]) => `${key}: ${value}`);
    if (parts.length > 0) {
      msg = `${msg}\n${parts.join("\n")}`;
    }
  }
  return msg;
}

export async function parseApiError(res: Response): Promise<string> {
  try {
    const data = (await res.json()) as {
      message?: string;
      error?: string;
      details?: Record<string, string>;
    };
    return formatApiErrorBody(data);
  } catch {
    return `Request failed (${res.status})`;
  }
}

export async function apiCall(path: string, options: RequestInit = {}): Promise<Response> {
  const url = path.startsWith("http") ? path : `${API_URL}${path.startsWith("/") ? path : `/${path}`}`;
  const headers = new Headers(options.headers);

  const token = await getAccessToken();
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
