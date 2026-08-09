import { ApiError } from "./errors";

export const API_BASE_URL =
  process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

const API_ORIGIN =
  typeof window !== "undefined" ? "" : API_BASE_URL;

let refreshPromise: Promise<boolean> | null = null;

// tryRefreshSession rotates the refresh token via the HttpOnly cookie.
// Concurrent 401s share a single refresh request.
async function tryRefreshSession(): Promise<boolean> {
  if (!refreshPromise) {
    refreshPromise = fetch(`${API_ORIGIN}/api/auth/refresh`, {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: "{}",
    })
      .then((response) => response.ok)
      .catch(() => false)
      .finally(() => {
        refreshPromise = null;
      });
  }

  return refreshPromise;
}

function redirectToLogin() {
  if (typeof window === "undefined") {
    return;
  }

  const segments = window.location.pathname.split("/");
  const locale = segments[1] === "en" ? "en" : "fa";

  if (window.location.pathname.includes("/console")) {
    window.location.assign(`/${locale}/login`);
  }
}

async function parseError(response: Response): Promise<ApiError> {
  let code: string | undefined;
  let message = `API request failed with status ${response.status}`;
  let fields: Record<string, string[]> | undefined;
  let requestId: string | undefined;

  try {
    const body = (await response.json()) as {
      error?: {
        code?: string;
        message?: string;
        fields?: Record<string, string[]>;
        request_id?: string;
      };
    };
    if (body?.error) {
      code = body.error.code;
      message = body.error.message ?? message;
      fields = body.error.fields;
      requestId = body.error.request_id;
    }
  } catch {
    // Non-JSON error body: keep defaults.
  }

  return new ApiError(message, response.status, code, fields, requestId);
}

async function rawRequest(path: string, init?: RequestInit): Promise<Response> {
  return fetch(`${API_ORIGIN}${path}`, {
    ...init,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...init?.headers,
    },
  });
}

// Auth endpoints must not trigger a transparent refresh: the refresh
// endpoint itself (recursion) and credential submissions (401 there means
// wrong credentials, not an expired session). /auth/me IS refreshable —
// it is the session probe used by the console auth gate.
function isRefreshableRequest(path: string): boolean {
  if (!path.startsWith("/api/auth/")) {
    return true;
  }
  return path === "/api/auth/me";
}

export async function apiRequest<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  let response: Response;

  try {
    response = await rawRequest(path, init);
  } catch {
    throw new ApiError("network_error", 0, "network_error");
  }

  // Expired access token: refresh once and retry transparently so users
  // never see a login prompt while their refresh token is valid.
  if (response.status === 401 && isRefreshableRequest(path)) {
    const refreshed = await tryRefreshSession();

    if (refreshed) {
      try {
        response = await rawRequest(path, init);
      } catch {
        throw new ApiError("network_error", 0, "network_error");
      }
    } else {
      redirectToLogin();
      throw await parseError(response);
    }

    if (response.status === 401) {
      redirectToLogin();
    }
  }

  if (!response.ok) {
    throw await parseError(response);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}
