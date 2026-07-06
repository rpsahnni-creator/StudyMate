import { NextResponse } from "next/server";
import { isTrustedOrigin } from "../../../../lib/csrf";

const REFRESH_COOKIE = "studyapp_refresh_token";
const REFRESH_MAX_AGE = 7 * 24 * 60 * 60; // 7 days — matches backend refresh TTL

export async function POST(request: Request) {
  if (!isTrustedOrigin(request)) {
    return NextResponse.json({ error: "Cross-origin request rejected" }, { status: 403 });
  }

  const body = (await request.json()) as { refresh_token?: string };
  const refreshToken = body.refresh_token?.trim();

  if (!refreshToken) {
    return NextResponse.json({ error: "refresh_token is required" }, { status: 400 });
  }

  const response = NextResponse.json({ ok: true });
  response.cookies.set(REFRESH_COOKIE, refreshToken, {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: REFRESH_MAX_AGE,
  });
  return response;
}

export async function DELETE(request: Request) {
  if (!isTrustedOrigin(request)) {
    return NextResponse.json({ error: "Cross-origin request rejected" }, { status: 403 });
  }

  const response = NextResponse.json({ ok: true });
  response.cookies.set(REFRESH_COOKIE, "", {
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: 0,
  });
  return response;
}
