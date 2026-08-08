"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { apiRequest } from "@/shared/api";
import type {
  AuthTokensResponse,
  AuthUser,
  OTPRequestResponse,
} from "@/shared/types/auth";

export function useMe() {
  return useQuery({
    queryKey: ["auth", "me"],
    queryFn: () => apiRequest<{ user: AuthUser }>("/api/v1/auth/me"),
    retry: false,
    staleTime: 60_000,
  });
}

export function useRequestOtp() {
  return useMutation({
    mutationFn: (phone: string) =>
      apiRequest<OTPRequestResponse>("/api/v1/auth/otp/request", {
        method: "POST",
        body: JSON.stringify({ phone }),
      }),
  });
}

export function useVerifyOtp() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: { phone: string; code: string }) =>
      apiRequest<AuthTokensResponse>("/api/v1/auth/otp/verify", {
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
      apiRequest<void>("/api/v1/auth/logout", {
        method: "POST",
        body: "{}",
      }),
    onSettled: () => {
      queryClient.clear();
    },
  });
}
