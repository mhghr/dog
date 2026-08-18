import { NextRequest } from "next/server";
import createMiddleware from "next-intl/middleware";

import { routing } from "./i18n/routing";
import { readSetCookies, type ParsedCookie } from "./shared/api/set-cookie";

const intlMiddleware = createMiddleware(routing);

const ACCESS_COOKIE = "mp_at";
const REFRESH_COOKIE = "mp_rt";

// Decodes the JWT exp claim without verifying the signature (the payload is
// plain base64url). Used to decide cheaply whether a refresh is needed instead
// of calling the API on every request.
function accessTokenExpired(token: string | undefined): boolean {
  if (!token) return false;
  try {
    const payload = JSON.parse(decodeBase64Url(token.split(".")[1] ?? ""));
    if (typeof payload?.exp === "number") {
      return payload.exp * 1000 <= Date.now();
    }
  } catch {
    // Unparseable token: let the API decide.
  }
  return false;
}

// Base64url (JWT) → UTF-8 string. atob handles standard base64 only, so the
// URL-safe alphabet must be translated first.
function decodeBase64Url(input: string): string {
  const normalized = input.replace(/-/g, "+").replace(/_/g, "/");
  const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, "=");
  return atob(padded);
}

// Rotates the session through the same-origin /api/auth/refresh endpoint
// (rewritten to the Go API) and returns the cookies it issues. Runs in the
// edge runtime, so it must not rely on server-only env vars.
async function refreshSession(
  cookieHeader: string,
  origin: string,
): Promise<ParsedCookie[] | null> {
  try {
    const response = await fetch(`${origin}/api/auth/refresh`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        cookie: cookieHeader,
      },
      body: "{}",
      cache: "no-store",
    });
    if (!response.ok) return null;
    return readSetCookies(response);
  } catch {
    return null;
  }
}

export default async function middleware(request: NextRequest) {
  const access = request.cookies.get(ACCESS_COOKIE)?.value;
  const refresh = request.cookies.get(REFRESH_COOKIE)?.value;

  let refreshed: ParsedCookie[] | null = null;

  // Refresh before the page renders so Server Components resolve a valid
  // access token (Next.js forbids writing cookies during render).
  if (refresh && (!access || accessTokenExpired(access))) {
    refreshed = await refreshSession(
      request.headers.get("cookie") ?? "",
      request.nextUrl.origin,
    );
  }

  let downstreamRequest: NextRequest = request;
  if (refreshed && refreshed.length > 0) {
    downstreamRequest = new NextRequest(request, {
      headers: new Headers(request.headers),
    });
    // The downstream Server Component reads name/value from the Cookie
    // header; attributes (HttpOnly etc.) are set on the response below.
    for (const cookie of refreshed) {
      downstreamRequest.cookies.set(cookie.name, cookie.value);
    }
  }

  const response = intlMiddleware(downstreamRequest);

  if (refreshed) {
    for (const cookie of refreshed) {
      response.cookies.set(cookie.name, cookie.value, {
        path: cookie.path,
        domain: cookie.domain,
        maxAge: cookie.maxAge,
        httpOnly: cookie.httpOnly,
        secure: cookie.secure,
        sameSite: cookie.sameSite,
      });
    }
  }

  return response;
}

export const config = {
  matcher: ["/((?!api|internal|events|_next|_vercel|.*\\..*).*)"],
};
