import { NextRequest, NextResponse } from "next/server";

const ACCESS_COOKIE = "mp_at";
const REFRESH_COOKIE = "mp_rt";

interface ExchangeResponse {
  access_token: string;
  access_token_expires_in: number;
  refresh_token: string;
  refresh_token_expires_in: number;
}

function loginRedirect(origin: string, locale: string) {
  return NextResponse.redirect(new URL(`/${locale}/login?error=oauth`, origin));
}

// Google redirects here after consent. The code is exchanged by the Go API
// (which holds the client secret); this handler only sets the HttpOnly
// session cookies and forwards the user into the console.
export async function GET(request: NextRequest) {
  const origin = request.nextUrl.origin;
  const code = request.nextUrl.searchParams.get("code");
  const stateRaw = request.nextUrl.searchParams.get("state") ?? "";
  const oauthError = request.nextUrl.searchParams.get("error");

  let locale = "fa";
  let nonce = "";

  try {
    const decoded = Buffer.from(stateRaw, "base64url").toString("utf8");
    const [statePart, localePart] = decoded.split(":");
    nonce = statePart ?? "";
    if (localePart === "en") {
      locale = "en";
    }
  } catch {
    return loginRedirect(origin, locale);
  }

  const stateCookie = request.cookies.get("oauth_state")?.value;

  if (oauthError || !code || !nonce || !stateCookie || stateCookie !== nonce) {
    return loginRedirect(origin, locale);
  }

  const apiBaseURL =
    process.env.API_INTERNAL_URL ??
    process.env.NEXT_PUBLIC_API_BASE_URL ??
    "http://localhost:8080";

  const redirectUri =
    process.env.GOOGLE_REDIRECT_URI ?? `${origin}/api/auth/callback/google`;

  let tokens: ExchangeResponse;

  try {
    const exchangeResponse = await fetch(
      `${apiBaseURL}/api/auth/google/exchange`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ code, redirect_uri: redirectUri }),
        cache: "no-store",
      },
    );

    if (!exchangeResponse.ok) {
      return loginRedirect(origin, locale);
    }

    tokens = (await exchangeResponse.json()) as ExchangeResponse;
  } catch {
    return loginRedirect(origin, locale);
  }

  const response = NextResponse.redirect(
    new URL(`/${locale}/app/dashboard`, origin),
  );

  const secure = process.env.NODE_ENV === "production";

  response.cookies.set(ACCESS_COOKIE, tokens.access_token, {
    httpOnly: true,
    sameSite: "lax",
    secure,
    path: "/",
    maxAge: tokens.access_token_expires_in,
  });
  response.cookies.set(REFRESH_COOKIE, tokens.refresh_token, {
    httpOnly: true,
    sameSite: "lax",
    secure,
    path: "/",
    maxAge: tokens.refresh_token_expires_in,
  });
  response.cookies.delete("oauth_state");

  return response;
}
