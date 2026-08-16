"use client";

import {
  createContext,
  useContext,
  useMemo,
  type ReactNode,
} from "react";

import { useLogout, useMe } from "@/platform/auth/use-auth";
import type { AuthState, AuthUser } from "@/shared/types/auth";

// Clerk-style client authentication API. The provider is seeded with the
// authentication state the server resolved (passed down from a Server
// Component), then keeps it fresh through the shared ["auth","me"] React
// Query entry — the same query useVerifyOtp and useLogout already write to,
// so there is exactly one source of truth on the client.
//
//   const { user, isLoaded, isSignedIn } = useAuth();
//   const { user } = useUser();

export interface AuthContextValue extends AuthState {
  logout: () => Promise<void>;
  refresh: () => Promise<AuthUser | null>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({
  initialAuth,
  children,
}: {
  initialAuth?: AuthState;
  children: ReactNode;
}) {
  const logoutMutation = useLogout();

  // initialData carries the server-resolved user into the first client render,
  // so the authenticated state in the server HTML survives hydration without
  // a second /auth/me round trip. A background 401 flips the state to
  // signed-out (session expiry) and useVerifyOtp rehydrates it after login.
  const meQuery = useMe({
    initialData:
      initialAuth?.isSignedIn && initialAuth.user
        ? { user: initialAuth.user }
        : undefined,
  });

  const value = useMemo<AuthContextValue>(() => {
    const failed = meQuery.isError && !meQuery.isFetching;

    const user = failed
      ? null
      : (meQuery.data?.user ?? initialAuth?.user ?? null);
    const isSignedIn = failed ? false : user !== null;

    const isLoaded =
      initialAuth?.isLoaded === true ||
      meQuery.isSuccess ||
      meQuery.isError;

    return {
      isLoaded,
      isSignedIn,
      user,
      session: initialAuth?.session ?? null,
      logout: async () => {
        await logoutMutation.mutateAsync();
      },
      refresh: async () => {
        const result = await meQuery.refetch();
        return result.data?.user ?? null;
      },
    };
  }, [meQuery, initialAuth, logoutMutation]);

  return (
    <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
  );
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within <AuthProvider>");
  }
  return ctx;
}

export function useUser() {
  const { user } = useAuth();
  return { user };
}
