"use client";

import { useMemo } from "react";
import { useAgents } from "@/hooks/use-agents";
import { useLocations } from "@/hooks/use-locations";
import type { ConnectionLine, MonitoringData, MonitoringStats, Probe, UserNode } from "@/types/monitoring";

const CITY_COORDS: Record<string, { lat: number; lng: number; country: string }> = {
  amsterdam: { lat: 52.3676, lng: 4.9041, country: "Netherlands" },
  frankfurt: { lat: 50.1109, lng: 8.6821, country: "Germany" },
  london: { lat: 51.5074, lng: -0.1278, country: "United Kingdom" },
  paris: { lat: 48.8566, lng: 2.3522, country: "France" },
  singapore: { lat: 1.3521, lng: 103.8198, country: "Singapore" },
  tokyo: { lat: 35.6762, lng: 139.6503, country: "Japan" },
  "new-york": { lat: 40.7128, lng: -74.006, country: "United States" },
  sydney: { lat: -33.8688, lng: 151.2093, country: "Australia" },
  sao: { lat: -23.5505, lng: -46.6333, country: "Brazil" },
  mumbai: { lat: 19.076, lng: 72.8777, country: "India" },
  dubai: { lat: 25.2048, lng: 55.2708, country: "UAE" },
  toronto: { lat: 43.6532, lng: -79.3832, country: "Canada" },
  tehran: { lat: 35.6892, lng: 51.389, country: "Iran" },
  fra: { lat: 50.1109, lng: 8.6821, country: "Germany" },
  ams: { lat: 52.3676, lng: 4.9041, country: "Netherlands" },
  sgp: { lat: 1.3521, lng: 103.8198, country: "Singapore" },
  nyc: { lat: 40.7128, lng: -74.006, country: "United States" },
  tyo: { lat: 35.6762, lng: 139.6503, country: "Japan" },
  "local-dev": { lat: 35.6892, lng: 51.389, country: "Iran" },
};

function resolveCoords(name: string, code: string) {
  const normalized = name.toLowerCase().replace(/\s+/g, "-");
  return CITY_COORDS[code.toLowerCase()]
    ?? CITY_COORDS[normalized]
    ?? CITY_COORDS["local-dev"];
}

function agentStatusToProbeStatus(status: string): Probe["status"] {
  switch (status) {
    case "active":
    case "approved":
      return "online";
    case "offline":
    case "draining":
    case "updating":
      return "warning";
    default:
      return "offline";
  }
}

const CENTER_COORDS = { lat: 35.6892, lng: 51.389 };
const AUTO_ID = "user-auto";

function buildProbes(
  locations: { id: string; name: string; code: string }[],
  agents: { id: string; location_id: string; name: string; status: string; hostname: string; latitude?: number | null; longitude?: number | null; city?: string; country?: string }[],
): Probe[] {
  if (locations.length === 0 && agents.length === 0) return [];

  const locMap = new Map(locations.map((l) => [l.id, l]));

  if (agents.length > 0) {
    return agents.map((agent) => {
      const loc = locMap.get(agent.location_id);

      let coords: { lat: number; lng: number };
      let city: string;
      let country: string;

      if (agent.latitude != null && agent.longitude != null) {
        coords = { lat: agent.latitude, lng: agent.longitude };
        city = agent.city || agent.hostname;
        country = agent.country || "";
      } else if (loc) {
        const resolved = resolveCoords(loc.name, loc.code);
        coords = { lat: resolved.lat, lng: resolved.lng };
        city = loc.name;
        country = resolved.country;
      } else {
        coords = CENTER_COORDS;
        city = agent.hostname;
        country = "";
      }

      return {
        id: agent.id,
        name: agent.name,
        country,
        city,
        latitude: coords.lat,
        longitude: coords.lng,
        status: agentStatusToProbeStatus(agent.status),
        latency: 0,
        packetLoss: 0,
      };
    });
  }

  return locations.map((loc) => {
    const coords = resolveCoords(loc.name, loc.code);
    return {
      id: loc.id,
      name: loc.name,
      country: coords.country,
      city: loc.name,
      latitude: coords.lat,
      longitude: coords.lng,
      status: "offline" as const,
      latency: 0,
      packetLoss: 0,
    };
  });
}

function buildConnections(probes: Probe[], userNodes: UserNode[]): ConnectionLine[] {
  if (probes.length === 0) return [];

  const targets = userNodes.length > 0
    ? userNodes.map((n) => ({ lat: n.latitude, lng: n.longitude }))
    : [{ lat: CENTER_COORDS.lat, lng: CENTER_COORDS.lng }];

  const lines: ConnectionLine[] = [];

  for (const probe of probes) {
    if (probe.status === "offline") continue;
    for (const target of targets) {
      lines.push({
        id: `conn-${probe.id}-${target.lat}-${target.lng}`,
        source: { lat: probe.latitude, lng: probe.longitude },
        target: { lat: target.lat, lng: target.lng },
        latency: probe.latency,
        status: probe.status,
      });
    }
  }

  return lines;
}

export function useMonitoring(): MonitoringData & { stats: MonitoringStats } {
  const agentsQuery = useAgents();
  const locationsQuery = useLocations();

  const loading = agentsQuery.isPending || locationsQuery.isPending;

  const probes = useMemo(
    () =>
      buildProbes(
        locationsQuery.data?.items ?? [],
        agentsQuery.data?.items ?? [],
      ),
    [locationsQuery.data, agentsQuery.data],
  );

  const userNodes = useMemo<UserNode[]>(() => [], []);

  const connections = useMemo(() => buildConnections(probes, userNodes), [probes, userNodes]);

  const stats = useMemo((): MonitoringStats => {
    const online = probes.filter((p) => p.status === "online").length;
    const latencies = connections.map((c) => c.latency).filter((l) => l > 0);
    const avgLatency = latencies.length > 0 ? Math.round(latencies.reduce((a, b) => a + b, 0) / latencies.length) : 0;
    return {
      totalProbes: probes.length,
      onlineProbes: online,
      avgLatency,
      activeConnections: connections.length,
    };
  }, [probes, connections]);

  return {
    probes,
    userNodes,
    connections,
    loading,
    stats,
  };
}
