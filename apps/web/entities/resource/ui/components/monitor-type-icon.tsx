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
