import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { apiRequest } from "@/lib/api-client";

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("apiRequest session refresh", () => {
  const fetchMock = vi.fn();

  beforeEach(() => {
    vi.stubGlobal("fetch", fetchMock);
    fetchMock.mockReset();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("refreshes and retries /auth/me after an expired access token", async () => {
    fetchMock.mockImplementation((input: RequestInfo | URL) => {
      const url = String(input);

      if (url.endsWith("/api/v1/auth/refresh")) {
        return Promise.resolve(jsonResponse(200, { access_token: "new" }));
      }

      if (url.endsWith("/api/v1/auth/me")) {
        const isRetry = fetchMock.mock.calls.some(([earlier]) =>
          String(earlier).endsWith("/api/v1/auth/refresh"),
        );
        if (isRetry) {
          return Promise.resolve(jsonResponse(200, { user: { id: "u1" } }));
        }
        return Promise.resolve(
          jsonResponse(401, { error: { code: "unauthorized", message: "expired" } }),
        );
      }

      return Promise.resolve(jsonResponse(404, {}));
    });

    await expect(
      apiRequest<{ user: { id: string } }>("/api/v1/auth/me"),
    ).resolves.toEqual({ user: { id: "u1" } });

    const calledPaths = fetchMock.mock.calls.map(([input]) => String(input));
    expect(calledPaths.some((url) => url.endsWith("/api/v1/auth/refresh"))).toBe(true);
  });

  it("never attempts refresh for the refresh endpoint itself", async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(401, { error: { code: "unauthorized", message: "no session" } }),
    );

    await expect(apiRequest("/api/v1/auth/refresh")).rejects.toMatchObject({
      status: 401,
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});
