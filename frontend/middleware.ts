import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const REFRESH_COOKIE = "studyapp_refresh_token";

const PROTECTED_PREFIXES = ["/scan", "/reports", "/admin", "/profile", "/plans/checkout", "/quiz"];

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  const isProtected = PROTECTED_PREFIXES.some(
    (prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`)
  );

  if (!isProtected) {
    return NextResponse.next();
  }

  const hasSession = Boolean(request.cookies.get(REFRESH_COOKIE)?.value);

  if (!hasSession) {
    const loginUrl = new URL("/auth/login", request.url);
    loginUrl.searchParams.set("from", pathname);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    "/scan/:path*",
    "/reports/:path*",
    "/admin/:path*",
    "/profile",
    "/profile/:path*",
    "/plans/checkout",
    "/quiz/:path*",
  ],
};
