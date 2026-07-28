export interface Probe {
  id: string;
  name: string;
  country: string;
  city: string;
  latitude: number;
  longitude: number;
  status: "online" | "warning" | "offline";
  latency: number;
  packetLoss: number;
}

export interface UserNode {
  id: string;
  username: string;
  latitude: number;
  longitude: number;
  ip: string;
  isp: string;
  city: string;
  country: string;
}

export interface ConnectionLine {
  id: string;
  source: { lat: number; lng: number };
  target: { lat: number; lng: number };
  latency: number;
  status: "online" | "warning" | "offline";
}

export interface MonitoringData {
  probes: Probe[];
  userNodes: UserNode[];
  connections: ConnectionLine[];
  loading: boolean;
}

export interface MonitoringStats {
  totalProbes: number;
  onlineProbes: number;
  avgLatency: number;
  activeConnections: number;
}
