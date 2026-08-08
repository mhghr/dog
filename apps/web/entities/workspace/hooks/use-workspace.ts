"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { workspaceApi } from "@/entities/workspace/api/workspace.api";

export function useWorkspaces() {
  return useQuery({
    queryKey: ["workspaces"],
    queryFn: () => workspaceApi.listWorkspaces(),
    placeholderData: (prev) => prev,
  });
}

export function useProjects() {
  return useQuery({
    queryKey: ["organization", "projects"],
    queryFn: () => workspaceApi.listProjects(),
  });
}

export function useCreateProject() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (name: string) => workspaceApi.createProject(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["organization", "projects"] });
    },
  });
}
