"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { apiRequest } from "@/lib/api-client";
import { toast } from "sonner";
import { useTranslations } from "next-intl";
import type { Project } from "@/types/organization";

export function useProjects() {
  return useQuery({
    queryKey: ["organization", "projects"],
    queryFn: () =>
      apiRequest<{ items: Project[] }>("/api/v1/organizations/projects"),
    select: (data) => data.items,
    placeholderData: (previous) => previous,
  });
}

export function useCreateProject() {
  const queryClient = useQueryClient();
  const tOrg = useTranslations("organization");

  return useMutation({
    mutationFn: (name: string) =>
      apiRequest<Project>("/api/v1/organizations/projects", {
        method: "POST",
        body: JSON.stringify({ name }),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["organization", "projects"] });
      toast.success(tOrg("projectCreated"));
    },
  });
}
