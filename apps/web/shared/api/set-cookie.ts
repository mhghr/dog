// Shared Set-Cookie parsing. Used by both the server-side API client
// (shared/api/server.ts) and the middleware (proxy.ts) so cookie attributes
// the Go API issues are applied identically in both contexts. This module is
// intentionally free of next/headers imports — middleware runs in the edge
// runtime where that module is unavailable.

export interface ParsedCookie {
  name: string;
  value: string;
  maxAge?: number;
  path?: string;
  domain?: string;
  secure?: boolean;
  httpOnly?: boolean;
  sameSite?: "lax" | "strict" | "none";
}

export function parseSetCookie(value: string): ParsedCookie | null {
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

// Reads every Set-Cookie header from a fetch response. getSetCookie() is
// available in modern runtimes (Node, edge); older environments fall back to
// splitting the combined header.
export function readSetCookies(response: Response): ParsedCookie[] {
  const raw =
    typeof response.headers.getSetCookie === "function"
      ? response.headers.getSetCookie()
      : (response.headers.get("set-cookie")?.split(",") ?? []);

  return raw
    .map((header) => parseSetCookie(header))
    .filter((cookie): cookie is ParsedCookie => cookie !== null);
}
