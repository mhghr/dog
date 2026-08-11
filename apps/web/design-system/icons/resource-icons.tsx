"use client";

import {
  Globe, Server, Database, Container, Network, Cloud,
  Monitor, Cpu, HardDrive, Router, Wifi, Shield,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";

type IconDef = {
  icon: LucideIcon;
  label: string;
  color: string;
};

export const RESOURCE_ICONS: Record<string, IconDef> = {
  server:          { icon: Server,    label: "Server",          color: "text-indigo-500 bg-indigo-500/10" },
  website:         { icon: Globe,     label: "Website",         color: "text-emerald-500 bg-emerald-500/10" },
  database:        { icon: Database,  label: "Database",        color: "text-violet-500 bg-violet-500/10" },
  container:       { icon: Container, label: "Container",       color: "text-cyan-500 bg-cyan-500/10" },
  "cloud-service": { icon: Cloud,     label: "Cloud Service",   color: "text-sky-500 bg-sky-500/10" },
  "network-device":{ icon: Network,   label: "Network Device",  color: "text-amber-500 bg-amber-500/10" },
  api:             { icon: Cpu,       label: "API",             color: "text-rose-500 bg-rose-500/10" },
  infrastructure:  { icon: Server,    label: "Infrastructure",  color: "text-blue-500 bg-blue-500/10" },
  web:             { icon: Globe,     label: "Web",             color: "text-emerald-500 bg-emerald-500/10" },
  network:         { icon: Network,   label: "Network",         color: "text-amber-500 bg-amber-500/10" },
  cloud:           { icon: Cloud,     label: "Cloud",           color: "text-sky-500 bg-sky-500/10" },
};

export const DEFAULT_RESOURCE_ICON: IconDef = {
  icon: Monitor,
  label: "Resource",
  color: "text-zinc-500 bg-zinc-500/10",
};

export function getResourceIcon(category?: string): IconDef {
  if (!category) return DEFAULT_RESOURCE_ICON;
  return RESOURCE_ICONS[category] ?? DEFAULT_RESOURCE_ICON;
}

export { Globe, Server, Database, Container, Network, Cloud, Monitor, Cpu, HardDrive, Router, Wifi, Shield };
export type { LucideIcon };
