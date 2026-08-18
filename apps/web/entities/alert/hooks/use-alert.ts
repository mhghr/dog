"use client";

import { useQuery } from "@tanstack/react-query";

import { alertApi } from "@/entities/alert/api/alert.api";

export function useAlerts() {
  return useQuery({
    queryKey: ["alerts"],
    queryFn: () => alertApi.listAlerts(),
    refetchInterval: 10_000,
    placeholderData: (previous) => previous,
  });
}
