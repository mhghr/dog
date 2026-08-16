"use client";

import { useQuery } from "@tanstack/react-query";

import { probeApi } from "@/entities/probe/api/probe.api";

export function useLocations() {
  return useQuery({
    queryKey: ["probe-locations"],
    queryFn: () => probeApi.listLocations(),
    refetchInterval: 30_000,
    placeholderData: (previous) => previous,
  });
}
