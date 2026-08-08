import { apiRequest } from "@/shared/api";
import { endpoints } from "@/shared/api/endpoints";

import type { StatusPage, StatusPageInput } from "@/features/status-pages/model/types";

export const statusPageApi = {
  list() {
    return apiRequest<{ items: StatusPage[] }>(endpoints.statusPage.list);
  },

  getById(id: string) {
    return apiRequest<StatusPage>(endpoints.statusPage.byId(id));
  },

  create(input: StatusPageInput) {
    return apiRequest<StatusPage>(endpoints.statusPage.list, {
      method: "POST",
      body: JSON.stringify(input),
    });
  },

  update(id: string, input: StatusPageInput) {
    return apiRequest<StatusPage>(endpoints.statusPage.byId(id), {
      method: "PUT",
      body: JSON.stringify(input),
    });
  },

  delete(id: string) {
    return apiRequest<void>(endpoints.statusPage.byId(id), {
      method: "DELETE",
    });
  },
};
