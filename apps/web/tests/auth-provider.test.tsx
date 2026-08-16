import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  act,
  renderHook,
  waitFor,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";

import { AuthProvider, useAuth } from "@/platform/auth/auth-provider";
import type { AuthState, AuthUser } from "@/shared/types/auth";

vi.mock("@/shared/api", () => ({
  apiRequest: vi.fn(),
}));

import { apiRequest } from "@/shared/api";

const mockApiRequest = apiRequest as ReturnType<typeof vi.fn>;

function renderAuth(initialAuth?: AuthState) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <AuthProvider initialAuth={initialAuth}>{children}</AuthProvider>
    </QueryClientProvider>
  );
  const utils = renderHook(() => useAuth(), { wrapper });
  return { utils, queryClient };
}

const user: AuthUser = {
  id: "u1",
  name: "Test User",
  email: "test@example.com",
  phone: "",
  avatar_url: "",
};

const signedInState: AuthState = {
  isLoaded: true,
  isSignedIn: true,
  user,
  session: { hasAccessToken: true, hasRefreshToken: true },
};

const signedOutState: AuthState = {
  isLoaded: true,
  isSignedIn: false,
  user: null,
  session: null,
};

beforeEach(() => {
  mockApiRequest.mockReset();
});

afterEach(() => {
  vi.clearAllMocks();
});

describe("AuthProvider hydration from server state", () => {
  it("exposes the server-resolved user immediately and does not refetch /me", () => {
    const { utils } = renderAuth(signedInState);

    expect(utils.result.current.isLoaded).toBe(true);
    expect(utils.result.current.isSignedIn).toBe(true);
    expect(utils.result.current.user?.id).toBe("u1");
    expect(utils.result.current.session?.hasAccessToken).toBe(true);
    // Server HTML and the first client render agree — no second auth request.
    expect(mockApiRequest).not.toHaveBeenCalled();
  });

  it("exposes the signed-out server verdict immediately", () => {
    const { utils } = renderAuth(signedOutState);

    expect(utils.result.current.isLoaded).toBe(true);
    expect(utils.result.current.isSignedIn).toBe(false);
    expect(utils.result.current.user).toBeNull();
  });

  it("stays signed out when the session probe fails", async () => {
    mockApiRequest.mockRejectedValue(new Error("unauthorized"));
    const { utils } = renderAuth(signedOutState);

    await waitFor(() => expect(utils.result.current.isLoaded).toBe(true));
    expect(utils.result.current.isSignedIn).toBe(false);
  });

  it("flips to signed out when the session can no longer be re-established", async () => {
    const { utils, queryClient } = renderAuth(signedInState);

    expect(utils.result.current.isSignedIn).toBe(true);

    mockApiRequest.mockRejectedValue(new Error("unauthorized"));

    // Simulate an expired session: the shared /auth/me query is revalidated
    // (window focus / invalidate) and the backend rejects it.
    await act(async () => {
      queryClient.invalidateQueries({ queryKey: ["auth", "me"] });
    });

    await waitFor(() => {
      expect(utils.result.current.isSignedIn).toBe(false);
    });
    expect(utils.result.current.user).toBeNull();
    expect(mockApiRequest).toHaveBeenCalled();
  });
});
