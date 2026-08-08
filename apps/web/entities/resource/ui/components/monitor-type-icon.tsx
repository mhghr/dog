import {
  Broadcast,
  CalendarCheck,
  Clock,
  EnvelopeSimple,
  Gauge,
  Globe,
  PlugsConnected,
  Pulse,
  ShieldCheck,
  SquaresFour,
  TreeStructure,
  Tray,
} from "@/shared/ui/icons";
import { cn } from "@/shared/utils/cn";

export function statusTone(status: string): string {
  switch (status) {
    case "active":
    case "up":
    case "ok":
      return "border-transparent bg-success/12 text-success";
    case "warning":
    case "degraded":
      return "border-transparent bg-warning/15 text-warning";
    case "down":
    case "error":
    case "failed":
      return "border-transparent bg-destructive/12 text-destructive";
    default:
      return "border-transparent bg-muted text-muted-foreground";
  }
}

const MONITOR_TYPE_ICONS: Record<string, typeof Pulse> = {
  ping: Broadcast,
  "http check": Globe,
  "tcp port": PlugsConnected,
  "dns resolution": TreeStructure,
  "ssl certificate": ShieldCheck,
  "domain expiry": CalendarCheck,
  "host metrics": Gauge,
  "docker monitor": Tray,
  "kubernetes monitor": SquaresFour,
  "database monitor": Tray,
  "smtp check": EnvelopeSimple,
  "ntp check": Clock,
};

export function MonitorTypeIcon({
  type,
  className,
}: {
  type: string;
  className?: string;
}) {
  const Icon = MONITOR_TYPE_ICONS[type.toLowerCase()] ?? Pulse;
  return <Icon className={cn("shrink-0", className)} aria-hidden />;
}
