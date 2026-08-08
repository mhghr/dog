"use client";

import { createContext, useContext, useEffect, useMemo } from "react";

import { useWorkspaces } from "@/entities/workspace/hooks/use-workspace";
import { useWorkspaceStore } from "@/platform/state";

interface WorkspaceContextValue {
  slug: string;
  id: string | null;
}

const WorkspaceContext = createContext<WorkspaceContextValue | null>(null);

export function useWorkspace() {
  const ctx = useContext(WorkspaceContext);
  if (!ctx) {
    throw new Error("useWorkspace must be used within a WorkspaceProvider");
  }
  return ctx;
}

// Resolves the workspace slug from the route to the backend workspace id and
// exposes it to every hook via the workspace store. Query keys are scoped by
// this id so a multi-tenant session never mixes workspaces.
export function WorkspaceProvider({
  slug,
  children,
}: {
  slug: string;
  children: React.ReactNode;
}) {
  const { data: workspaces } = useWorkspaces();
  const { id, setWorkspace, clear } = useWorkspaceStore();

  useEffect(() => {
    const workspace = workspaces?.items.find((w) => w.slug === slug);
    setWorkspace(slug, workspace?.id ?? null);
    return clear;
  }, [workspaces, slug, setWorkspace, clear]);

  const value = useMemo(() => ({ slug, id }), [slug, id]);

  return (
    <WorkspaceContext.Provider value={value}>
      {children}
    </WorkspaceContext.Provider>
  );
}
