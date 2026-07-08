import { API_URL } from "./config";

const REACHABILITY_TIMEOUT_MS = 8000;

/** Quick health check before starting a multi-page upload batch. */
export async function assertServerReachable(): Promise<void> {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), REACHABILITY_TIMEOUT_MS);
  try {
    const res = await fetch(`${API_URL}/health`, {
      method: "GET",
      signal: controller.signal,
    });
    if (!res.ok) {
      throw new Error(`Server health check failed (${res.status})`);
    }
  } catch (error) {
    throw new Error(formatNetworkError(error));
  } finally {
    clearTimeout(timer);
  }
}

export function formatNetworkError(error: unknown): string {
  const raw = error instanceof Error ? error.message : String(error);
  const normalized = raw.toLowerCase();

  if (
    normalized.includes("connectexception") ||
    normalized.includes("failed to connect") ||
    normalized.includes("network request failed") ||
    normalized.includes("econnrefused") ||
    normalized.includes("connection refused") ||
    normalized.includes("unable to resolve host") ||
    normalized.includes("aborted") ||
    normalized.includes("cancel") ||
    normalized.includes("fetch failed") ||
    normalized.includes("timeout")
  ) {
    return (
      `Server tak connect nahi ho paaya (${API_URL}).\n\n` +
      "Check karein:\n" +
      "• PC par backend chal raha ho (go run ./cmd/api)\n" +
      "• Phone aur PC same Wi‑Fi par hon\n" +
      "• mobile/.env mein sahi PC IP ho (ipconfig → IPv4)\n" +
      "• Windows Firewall port 8080 allow ho"
    );
  }

  return raw || "Network error";
}

export function isRetryableNetworkError(error: unknown): boolean {
  const msg = (error instanceof Error ? error.message : String(error)).toLowerCase();
  return (
    msg.includes("connect") ||
    msg.includes("network") ||
    msg.includes("timeout") ||
    msg.includes("aborted") ||
    msg.includes("cancel") ||
    msg.includes("fetch failed") ||
    msg.includes("socket") ||
    msg.includes("econnrefused")
  );
}
