import { apiRequest } from "@/shared/api";
import { endpoints } from "@/shared/api/endpoints";
import type {
  ProbeLocation,
  ProbeLocationInput,
} from "@/entities/probe/model/types";
import type { DashboardSummary, LocationListResponse } from "@/shared/types/api";

export const probeApi = {
  listLocations() {
    return apiRequest<LocationListResponse>(endpoints.probe.locations);
  },

  createLocation(input: ProbeLocationInput) {
    return apiRequest<ProbeLocation>(endpoints.probe.locations, {
      method: "POST",
      body: JSON.stringify(input),
    });
  },

  updateLocation(id: string, input: ProbeLocationInput) {
    return apiRequest<ProbeLocation>(`${endpoints.probe.locations}/${id}`, {
      method: "PUT",
      body: JSON.stringify(input),
    });
  },

  deleteLocation(id: string) {
    return apiRequest<void>(`${endpoints.probe.locations}/${id}`, {
      method: "DELETE",
    });
  },

  getMonitoringSummary() {
    return apiRequest<DashboardSummary>(endpoints.dashboard.summary);
  },
};
