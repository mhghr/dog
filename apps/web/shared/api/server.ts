import { cookies } from "next/headers";

import type { AuthSessionInfo, AuthUser } from "@/shared/types/auth";

// Server-side API client. Runs only in the Next.js server runtime and
// mirrors the browser client (shared/api/client.ts) so both resolve the same
// backend endpoints: the request's HttpOnly cookies are forwarded to the Go
// API, and a failed /auth/me is transparently retried after rotating the
// refresh token. Any new cookies the backend issues are applied to the
// outgoing response so the browser and the server stay in sync.

const ACCESS_COOKIE = "mp_at";
const REFRESH_COOKIE = "mp_rt";

const API_INTERNAL_URL =
  process.env.API_INTERNAL_URL ??
  process.env.NEXT_PUBLIC_API_BASE_URL ??
  "http://localhost:5000";

async function buildHeaders(): Promise<HeadersInit> {
  const store = await cookies();
  const cookieHeader = store.toString();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (cookieHeader) {
    headers.cookie = cookieHeader;
  }
  return headers;
}

interface ParsedCookie {
  name: string;
  value: string;
  maxAge?: number;
  path?: string;
  domain?: string;
  secure?: boolean;
  httpOnly?: boolean;
  sameSite?: "lax" | "strict" | "none";
}

function parseSetCookie(value: string): ParsedCookie | null {
  const parts = value.split(";").map((part) => part.trim());
  const [nameValue, ...attributes] = parts;
  const separator = nameValue.indexOf("=");
  if (separator <= 0) {
    return null;
  }

  const cookie: ParsedCookie = {
    name: nameValue.slice(0, separator),
    value: nameValue.slice(separator + 1),
  };

  for (const attribute of attributes) {
    const [rawKey, rawValue = ""] = attribute.split("=");
    const key = rawKey.toLowerCase();
    const value = rawValue.trim();

    switch (key) {
      case "max-age":
        cookie.maxAge = Number.parseInt(value, 10);
        break;
      case "path":
        cookie.path = value;
        break;
      case "domain":
        cookie.domain = value;
        break;
      case "secure":
        cookie.secure = true;
        break;
      case "httponly":
        cookie.httpOnly = true;
        break;
      case "samesite":
        cookie.sameSite = value.toLowerCase() as ParsedCookie["sameSite"];
        break;
    }
  }

  return cookie;
}

// Applies cookies the backend set during a server-side refresh to the
// outgoing Next.js response so the browser keeps a valid session.
async function applySetCookies(response: Response): Promise<void> {
  const store = await cookies();
  const raw = (
    typeof response.headers.getSetCookie === "function"
      ? response.headers.getSetCookie()
      : response.headers.get("set-cookie")?.split(",") ?? []
  ) as string[];

  for (const header of raw) {
    const parsed = parseSetCookie(header);
    if (!parsed) {
      continue;
    }

    if (parsed.maxAge !== undefined && parsed.maxAge <= 0) {
      store.delete(parsed.name);
      continue;
    }

    store.set(parsed.name, parsed.value, {
      httpOnly: parsed.httpOnly,
      secure: parsed.secure,
      sameSite: parsed.sameSite,
      path: parsed.path,
      domain: parsed.domain,
      maxAge: parsed.maxAge,
    });
  }
}

export class ServerApiError extends Error {
  readonly status: number;
  readonly path: string;

  constructor(status: number, path: string) {
    super(`Server API request failed: ${status} ${path}`);
    this.name = "ServerApiError";
    this.status = status;
    this.path = path;
  }
}

export async function serverApiRequest<T>(
  path: string,
  init?: RequestInit,
  opts?: { refreshOn401?: boolean },
): Promise<T> {
  const doFetch = async () =>
    fetch(`${API_INTERNAL_URL}${path}`, {
      ...init,
      headers: {
        ...(init?.headers as Record<string, string> | undefined),
        ...(await buildHeaders()),
      },
      cache: "no-store",
    });

  let response = await doFetch();

  // The access token may have expired between the request's auth check and
  // this data call. Rotate the refresh token once and retry, exactly like the
  // browser client (shared/api/client.ts) does.
  if (response.status === 401 && opts?.refreshOn401 && (await refreshSession())) {
    response = await doFetch();
  }

  if (!response.ok) {
    throw new ServerApiError(response.status, path);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

interface MeResponse {
  user: AuthUser;
}

async function fetchCurrentUser(): Promise<AuthUser | null> {
  const response = await fetch(`${API_INTERNAL_URL}/api/auth/me`, {
    headers: await buildHeaders(),
    cache: "no-store",
  });

  if (!response.ok) {
    return null;
  }

  const body = (await response.json()) as MeResponse;
  return body?.user ?? null;
}

async function refreshSession(): Promise<boolean> {
  const response = await fetch(`${API_INTERNAL_URL}/api/auth/refresh`, {
    method: "POST",
    headers: await buildHeaders(),
    body: "{}",
    cache: "no-store",
  });

  if (!response.ok) {
    return false;
  }

  await applySetCookies(response);
  return true;
}

// Resolves the authenticated user for the current request. Returns null when
// the request carries no session cookies or the session cannot be
// re-established. Used by the server-side auth layer (getAuth/getCurrentUser).
export async function serverResolveAuth(): Promise<{
  user: AuthUser | null;
  session: AuthSessionInfo;
}> {
  const store = await cookies();
  const hasAccessToken = store.has(ACCESS_COOKIE);
  const hasRefreshToken = store.has(REFRESH_COOKIE);

  const session: AuthSessionInfo = {
    hasAccessToken,
    hasRefreshToken,
  };

  if (!hasAccessToken && !hasRefreshToken) {
    return { user: null, session };
  }

  let user = await fetchCurrentUser();

  // Access token expired: rotate the refresh token once and retry, exactly
  // like the browser client does.
  if (!user && hasRefreshToken) {
    if (await refreshSession()) {
      user = await fetchCurrentUser();
    }
  }

  return { user, session };
}
