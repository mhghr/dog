export interface AuthUser {
  id: string;
  name: string;
  email: string;
  phone: string;
  avatar_url: string;
  roles?: string[];
}

export interface AuthTokensResponse {
  user: AuthUser;
  access_token: string;
  access_token_expires_in: number;
  refresh_token: string;
  refresh_token_expires_in: number;
}

// Server-side facts about the current session. The backend owns the real
// session; this only mirrors the observable cookie state the server used to
// resolve the user, so the client never has to guess.
export interface AuthSessionInfo {
  hasAccessToken: boolean;
  hasRefreshToken: boolean;
}

// Normalized authentication state shared by the server and client auth
// layers. Server Components produce it via getAuth(); the client AuthProvider
// hydrates from it and keeps it fresh through the ["auth","me"] query.
export interface AuthState {
  isLoaded: boolean;
  isSignedIn: boolean;
  user: AuthUser | null;
  session: AuthSessionInfo | null;
}

export const EMPTY_AUTH_STATE: AuthState = {
  isLoaded: false,
  isSignedIn: false,
  user: null,
  session: null,
};

export interface OTPRequestResponse {
  status: string;
  retry_after: number;
  dev_code?: string;
}
