"use client";

import { useQuery } from "@tanstack/react-query";
import { workspaceApi } from "@/entities/workspace/api/workspace.api";

export function useWorkspaces() {
  return useQuery({
    queryKey: ["workspaces"],
    queryFn: () => workspaceApi.list(),
    placeholderData: (prev) => prev,
  });
}
