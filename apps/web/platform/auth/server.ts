import { cache } from "react";

import { serverResolveAuth } from "@/shared/api/server";
import type { AuthState, AuthSessionInfo, AuthUser } from "@/shared/types/auth";

// Server-side authentication API (Clerk-style `auth()` / `getCurrentUser()`).
// Use from Server Components, Server Actions and Route Handlers:
//
//   const user = await getCurrentUser();
//   const { user, isSignedIn } = await getAuth();
//
// Resolution is request-scoped: cookies are forwarded to the authoritative Go
// API and the result is memoized per request via React's cache(), so a Server
// Component tree never resolves the session more than once. This works
// without any client hydration.

export const getAuth = cache(async (): Promise<AuthState> => {
  const { user, session } = await serverResolveAuth();

  return {
    isLoaded: true,
    isSignedIn: user !== null,
    user,
    session,
  };
});

export const getCurrentUser = cache(
  async (): Promise<AuthUser | null> => {
    const { user } = await getAuth();
    return user;
  },
);

// Request-scoped session facts (which tokens are present), used when callers
// need to distinguish "signed out" from "token expired, refresh available".
export const getSessionInfo = cache(
  async (): Promise<AuthSessionInfo | null> => {
    const { session } = await getAuth();
    return session;
  },
);
