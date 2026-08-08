"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { probeApi } from "@/entities/probe/api/probe.api";
import type { ProbeLocationInput } from "@/entities/probe/model/types";
import type { LocationListResponse } from "@/shared/types/api";

export function useLocations() {
  return useQuery({
    queryKey: ["probe-locations"],
    queryFn: () => probeApi.listLocations(),
    refetchInterval: 30_000,
    placeholderData: (previous) => previous,
  });
}

export function useCreateLocation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: ProbeLocationInput) => probeApi.createLocation(input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["probe-locations"] });
    },
  });
}

export function useUpdateLocation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, ...input }: ProbeLocationInput & { id: string }) =>
      probeApi.updateLocation(id, input),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["probe-locations"] });
    },
  });
}

export function useDeleteLocation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => probeApi.deleteLocation(id),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["probe-locations"] });
    },
  });
}
