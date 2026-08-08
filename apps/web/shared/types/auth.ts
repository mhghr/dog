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

export interface OTPRequestResponse {
  status: string;
  retry_after: number;
  dev_code?: string;
}
