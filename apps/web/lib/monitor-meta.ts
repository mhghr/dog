import type { MonitorStatus, MonitorType } from "@/types/monitor";
import type { AppIcon } from "@/lib/icons";
import {
  Broadcast,
  CalendarCheck,
  Clock,
  EnvelopeSimple,
  Globe,
  PlugsConnected,
  ShieldCheck,
  TreeStructure,
} from "@/lib/icons";

export const MONITOR_TYPES: MonitorType[] = [
  "http",
  "tcp",
  "dns",
  "ping",
  "tls",
  "domain_expiration",
  "smtp",
  "ntp",
];

export const MONITOR_TYPE_ICONS: Record<MonitorType, AppIcon> = {
  http: Globe,
  ping: Broadcast,
  tcp: PlugsConnected,
  dns: TreeStructure,
  tls: ShieldCheck,
  domain_expiration: CalendarCheck,
  smtp: EnvelopeSimple,
  ntp: Clock,
};

export interface MonitorTypeGroup {
  key: "web" | "network" | "domain_email";
  types: MonitorType[];
}

export const MONITOR_TYPE_GROUPS: MonitorTypeGroup[] = [
  { key: "web", types: ["http", "tls"] },
  { key: "network", types: ["ping", "tcp", "dns", "ntp"] },
  { key: "domain_email", types: ["domain_expiration", "smtp"] },
];

export const DEFAULT_INTERVALS: Record<MonitorType, number> = {
  http: 60,
  tcp: 60,
  ping: 60,
  dns: 60,
  tls: 3600,
  domain_expiration: 43200,
  smtp: 60,
  ntp: 300,
};

export const MIN_INTERVALS: Record<MonitorType, number> = {
  http: 10,
  tcp: 10,
  ping: 10,
  dns: 30,
  tls: 300,
  domain_expiration: 3600,
  smtp: 30,
  ntp: 60,
};

export const STATUS_STYLES: Record<
  MonitorStatus,
  { badge: string; dot: string }
> = {
  up: {
    badge: "border-transparent bg-success/12 text-success",
    dot: "bg-success",
  },
  down: {
    badge: "border-transparent bg-destructive/12 text-destructive",
    dot: "bg-destructive",
  },
  unknown: {
    badge: "border-transparent bg-muted text-muted-foreground",
    dot: "bg-muted-foreground/60",
  },
  paused: {
    badge: "border-transparent bg-warning/15 text-warning",
    dot: "bg-warning",
  },
};
