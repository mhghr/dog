"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { apiRequest } from "@/shared/api";
import type {
  AuthTokensResponse,
  AuthUser,
  OTPRequestResponse,
} from "@/shared/types/auth";

export function useMe(opts?: { enabled?: boolean }) {
  return useQuery({
    queryKey: ["auth", "me"],
    queryFn: () => apiRequest<{ user: AuthUser }>("/api/auth/me"),
    retry: false,
    staleTime: 60_000,
    enabled: opts?.enabled !== false,
  });
}

export function useRequestOtp() {
  return useMutation({
    mutationFn: (phone: string) =>
      apiRequest<OTPRequestResponse>("/api/auth/otp/request", {
        method: "POST",
        body: JSON.stringify({ phone }),
      }),
  });
}

export function useVerifyOtp() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: { phone: string; code: string }) =>
      apiRequest<AuthTokensResponse>("/api/auth/otp/verify", {
        method: "POST",
        body: JSON.stringify(input),
      }),
    onSuccess: (data) => {
      queryClient.setQueryData(["auth", "me"], { user: data.user });
    },
  });
}

export function useLogout() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () =>
      apiRequest<void>("/api/auth/logout", {
        method: "POST",
        body: "{}",
      }),
    onSettled: () => {
      queryClient.clear();
    },
  });
}
