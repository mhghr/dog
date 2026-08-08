"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { statusPageApi } from "@/features/status-pages/api/status-page.api";
import type { StatusPage, StatusPageInput } from "@/features/status-pages/model/types";

export function useStatusPages() {
  return useQuery({
    queryKey: ["status-pages"],
    queryFn: () => statusPageApi.list(),
    refetchInterval: 30_000,
    placeholderData: (previous) => previous,
  });
}

export function useStatusPage(statusPageId: string) {
  return useQuery({
    queryKey: ["status-pages", statusPageId],
    queryFn: () => statusPageApi.getById(statusPageId),
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
    mutationFn: (input: StatusPageInput) => statusPageApi.create(input),
    onSuccess: () => invalidate(),
  });
}

export function useUpdateStatusPage(statusPageId: string) {
  const invalidate = useInvalidateStatusPages();

  return useMutation({
    mutationFn: (input: StatusPageInput) =>
      statusPageApi.update(statusPageId, input),
    onSuccess: () => invalidate(statusPageId),
  });
}

export function useDeleteStatusPage() {
  const invalidate = useInvalidateStatusPages();

  return useMutation({
    mutationFn: (statusPageId: string) => statusPageApi.delete(statusPageId),
    onSuccess: () => invalidate(),
  });
}
