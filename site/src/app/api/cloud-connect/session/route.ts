import { NextResponse } from "next/server";
import {
  CLOUD_SESSION_COOKIE,
  encodeCloudSession,
} from "@/lib/cloud-session";

export const runtime = "nodejs";

function cookieOptions() {
  return {
    httpOnly: true,
    sameSite: "lax" as const,
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 60 * 60 * 24 * 30,
    domain: process.env.CLOUD_SESSION_COOKIE_DOMAIN || undefined,
  };
}

function safeRedirect(request: Request, rawRedirect: string | null) {
  const fallback = "/cloud";
  if (!rawRedirect || !rawRedirect.startsWith("/")) {
    return new URL(fallback, request.url);
  }

  return new URL(rawRedirect, request.url);
}

export async function POST(request: Request) {
  const formData = await request.formData();

  const intent = String(formData.get("intent") ?? "login").trim();
  const redirectTo = String(formData.get("redirectTo") ?? "/cloud").trim();
  const redirectUrl = safeRedirect(request, redirectTo);

  if (intent === "logout") {
    const response = NextResponse.redirect(redirectUrl, { status: 303 });
    response.cookies.set(CLOUD_SESSION_COOKIE, "", {
      ...cookieOptions(),
      maxAge: 0,
    });
    return response;
  }

  const name = String(formData.get("name") ?? "").trim();
  const email = String(formData.get("email") ?? "").trim().toLowerCase();
  const role = String(formData.get("role") ?? "member").trim().toLowerCase();

  if (!name || !email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
    const invalidUrl = safeRedirect(
      request,
      `/cloud/login?invalid=1&redirectTo=${encodeURIComponent(redirectTo)}`,
    );
    return NextResponse.redirect(invalidUrl, { status: 303 });
  }

  const response = NextResponse.redirect(redirectUrl, { status: 303 });
  response.cookies.set(
    CLOUD_SESSION_COOKIE,
    encodeCloudSession({ name, email, role }),
    cookieOptions(),
  );

  return response;
}
