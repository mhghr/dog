"use client";

import * as React from "react";
import { createContext, useContext } from "react";

import { useProjects } from "@/hooks/use-organization";
import type { Project } from "@/types/organization";

interface ProjectContextValue {
  projectId: string | null;
  setProjectId: (id: string) => void;
  projects: Project[];
  isLoading: boolean;
}

const ProjectContext = createContext<ProjectContextValue | null>(null);

export function useProjectContext() {
  const ctx = useContext(ProjectContext);
  if (!ctx) {
    throw new Error("useProjectContext must be used within a ProjectProvider");
  }
  return ctx;
}

function getStoredProjectId(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("selectedProjectId");
}

function setStoredProjectId(id: string) {
  localStorage.setItem("selectedProjectId", id);
}

export function ProjectProvider({ children }: { children: React.ReactNode }) {
  const { data: projects = [], isLoading } = useProjects();
  const [projectId, setProjectIdState] = React.useState<string | null>(
    getStoredProjectId,
  );

  const setProjectId = React.useCallback((id: string) => {
    setStoredProjectId(id);
    setProjectIdState(id);
  }, []);

  const value = React.useMemo<ProjectContextValue>(
    () => ({ projectId, setProjectId, projects, isLoading }),
    [projectId, setProjectId, projects, isLoading],
  );

  return (
    <ProjectContext.Provider value={value}>{children}</ProjectContext.Provider>
  );
}
