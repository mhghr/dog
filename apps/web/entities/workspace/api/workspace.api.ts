import { apiRequest } from "@/shared/api";
import { endpoints } from "@/shared/api/endpoints";
import type { Project, Workspace } from "@/entities/workspace/model/types";

export const workspaceApi = {
  listWorkspaces() {
    return apiRequest<{ items: Workspace[] }>(endpoints.workspace.list);
  },

  listProjects() {
    return apiRequest<{ items: Project[] }>(endpoints.organization.projects);
  },

  createProject(name: string) {
    return apiRequest<Project>(endpoints.organization.projects, {
      method: "POST",
      body: JSON.stringify({ name }),
    });
  },
};
