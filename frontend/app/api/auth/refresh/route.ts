import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import { isTrustedOrigin } from "../../../../lib/csrf";

const REFRESH_COOKIE = "studyapp_refresh_token";
const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export async function POST(request: Request) {
  if (!isTrustedOrigin(request)) {
    return NextResponse.json({ message: "Cross-origin request rejected" }, { status: 403 });
  }

  const cookieStore = await cookies();
  const refreshToken = cookieStore.get(REFRESH_COOKIE)?.value;

  if (!refreshToken) {
    return NextResponse.json({ message: "No refresh token" }, { status: 401 });
  }

  const backendRes = await fetch(`${API_URL}/auth/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });

  const data = await backendRes.json();

  if (!backendRes.ok) {
    const response = NextResponse.json(data, { status: backendRes.status });
    response.cookies.set(REFRESH_COOKIE, "", {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      maxAge: 0,
    });
    return response;
  }

  const response = NextResponse.json(data, { status: 200 });

  if (data.refresh_token) {
    response.cookies.set(REFRESH_COOKIE, data.refresh_token, {
      httpOnly: true,
      secure: process.env.NODE_ENV === "production",
      sameSite: "lax",
      path: "/",
      maxAge: 7 * 24 * 60 * 60,
    });
  }

  return response;
}
