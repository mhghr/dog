"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";

import { apiRequest } from "@/lib/api-client";

interface LocationInput {
  name: string;
  code: string;
}

interface Location {
  id: string;
  name: string;
  code: string;
  enabled: boolean;
  created_at: string;
}

export function useCreateLocation() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (input: LocationInput) =>
      apiRequest<Location>("/api/v1/probe-locations", {
        method: "POST",
        body: JSON.stringify(input),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["probe-locations"] });
    },
  });
}
