"use client";

import { useQuery } from "@tanstack/react-query";

import { apiRequest } from "@/lib/api-client";
import type { LocationListResponse } from "@/types/api";

export function useLocations() {
  return useQuery({
    queryKey: ["probe-locations"],
    queryFn: () => apiRequest<LocationListResponse>("/api/v1/probe-locations"),
    refetchInterval: 30_000,
  });
}
