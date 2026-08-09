import { apiRequest } from "@/shared/api";
import { endpoints } from "@/shared/api/endpoints";
import type { Workspace } from "@/entities/workspace/model/types";

export const workspaceApi = {
  list() {
    return apiRequest<{ items: Workspace[] }>(endpoints.workspace.list);
  },

  create(name: string) {
    return apiRequest<Workspace>(endpoints.workspace.list, {
      method: "POST",
      body: JSON.stringify({ name }),
    });
  },
};
