"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { apiRequest } from "@/lib/api-client";
import type { StatusPage, StatusPageInput } from "@/types/status-page";

export function useStatusPages() {
  return useQuery({
    queryKey: ["status-pages"],
    queryFn: () => apiRequest<{ items: StatusPage[] }>("/api/v1/status-pages"),
    refetchInterval: 30_000,
  });
}

export function useStatusPage(statusPageId: string) {
  return useQuery({
    queryKey: ["status-pages", statusPageId],
    queryFn: () => apiRequest<StatusPage>(`/api/v1/status-pages/${statusPageId}`),
    enabled: Boolean(statusPageId),
  });
}

function useInvalidateStatusPages() {
  const queryClient = useQueryClient();

  return async (statusPageId?: string) => {
    await queryClient.invalidateQueries({ queryKey: ["status-pages"] });
    if (statusPageId) {
      await queryClient.invalidateQueries({
        queryKey: ["status-pages", statusPageId],
      });
    }
  };
}

export function useCreateStatusPage() {
  const invalidate = useInvalidateStatusPages();

  return useMutation({
    mutationFn: (input: StatusPageInput) =>
      apiRequest<StatusPage>("/api/v1/status-pages", {
        method: "POST",
        body: JSON.stringify(input),
      }),
    onSuccess: () => invalidate(),
  });
}

export function useUpdateStatusPage(statusPageId: string) {
  const invalidate = useInvalidateStatusPages();

  return useMutation({
    mutationFn: (input: StatusPageInput) =>
      apiRequest<StatusPage>(`/api/v1/status-pages/${statusPageId}`, {
        method: "PUT",
        body: JSON.stringify(input),
      }),
    onSuccess: () => invalidate(statusPageId),
  });
}

export function useDeleteStatusPage() {
  const invalidate = useInvalidateStatusPages();

  return useMutation({
    mutationFn: (statusPageId: string) =>
      apiRequest<void>(`/api/v1/status-pages/${statusPageId}`, {
        method: "DELETE",
      }),
    onSuccess: () => invalidate(),
  });
}
