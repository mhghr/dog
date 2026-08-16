import { apiRequest } from "@/shared/api";
import { endpoints } from "@/shared/api/endpoints";
import type { LocationListResponse } from "@/shared/types/api";
import type { DashboardSummary } from "@/shared/types/api";

export interface GeoLocation {
  country: string;
  city: string;
  lat: number;
  lon: number;
}

export const probeApi = {
  listLocations() {
    return apiRequest<LocationListResponse>(endpoints.probe.locations);
  },

  geoIpLookup(ip: string) {
    return apiRequest<GeoLocation>(endpoints.geoip(ip));
  },

  getMonitoringSummary() {
    return apiRequest<DashboardSummary>(endpoints.dashboard.summary);
  },
};
