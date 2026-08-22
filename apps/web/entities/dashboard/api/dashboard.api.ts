import { apiRequest } from "@/shared/api";
import { endpoints } from "@/shared/api/endpoints";
import type { DashboardSummary } from "@/entities/dashboard/model/types";

export const dashboardApi = {
  summary() {
    return apiRequest<DashboardSummary>(endpoints.dashboard.summary);
  },
};
