// Minimal same-origin check for Next.js Route Handlers that mutate session
// cookies. Browsers attach an `Origin` header to cross-site fetch/XHR
// requests (and to same-site POSTs in modern browsers); comparing it against
// the request's own origin blocks cross-site script/form submissions from
// silently setting or rotating our httpOnly session cookie (CSRF).
//
// Note: some legitimate same-origin requests (plain top-level navigations,
// very old browsers) can omit `Origin`. We only reject when the header is
// present and mismatched, so we never break normal same-site usage while
// still stopping the common CSRF case (a script on another origin issuing
// a fetch/XHR with credentials).
export function isTrustedOrigin(request: Request): boolean {
  const origin = request.headers.get("origin");
  if (!origin) return true;

  try {
    const requestOrigin = new URL(request.url).origin;
    return origin === requestOrigin;
  } catch {
    return false;
  }
}
