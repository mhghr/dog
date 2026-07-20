import { NextRequest, NextResponse } from "next/server";

const GOOGLE_AUTH_URL = "https://accounts.google.com/o/oauth2/v2/auth";

// Starts the Google OAuth code flow. The redirect URI must exactly match the
// URI registered in the Google console.
export function GET(request: NextRequest) {
  const clientId = process.env.GOOGLE_CLIENT_ID;
  const redirectUri =
    process.env.GOOGLE_REDIRECT_URI ??
    `${request.nextUrl.origin}/api/auth/callback/google`;

  const locale = request.nextUrl.searchParams.get("locale") === "en" ? "en" : "fa";

  if (!clientId) {
    return NextResponse.redirect(
      new URL(`/${locale}/login?error=oauth`, request.nextUrl.origin),
    );
  }

  const nonce = crypto.randomUUID();
  const state = Buffer.from(`${nonce}:${locale}`).toString("base64url");

  const authURL = new URL(GOOGLE_AUTH_URL);
  authURL.searchParams.set("client_id", clientId);
  authURL.searchParams.set("redirect_uri", redirectUri);
  authURL.searchParams.set("response_type", "code");
  authURL.searchParams.set("scope", "openid email profile");
  authURL.searchParams.set("state", state);
  authURL.searchParams.set("prompt", "select_account");

  const response = NextResponse.redirect(authURL);
  response.cookies.set("oauth_state", nonce, {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/api/auth",
    maxAge: 600,
  });

  return response;
}
