import {
  Broadcast,
  Browser,
  CalendarCheck,
  Clock,
  EnvelopeSimple,
  Gauge,
  Globe,
  Monitor,
  PlugsConnected,
  Pulse,
  ShieldCheck,
  ShieldWarning,
  SquaresFour,
  Tray,
  TreeStructure,
} from "@/shared/ui/icons";
import { cn } from "@/shared/utils/cn";

const RESOURCE_TYPE_ICONS: Record<string, typeof Globe> = {
  server: Monitor,
  "virtual machine": Monitor,
  "virtual-machine": Monitor,
  website: Globe,
  "api endpoint": Globe,
  "api-endpoint": Globe,
  database: Tray,
  "docker host": Tray,
  "docker-host": Tray,
  "kubernetes cluster": SquaresFour,
  "kubernetes-cluster": SquaresFour,
  "network device": Broadcast,
  "network-device": Broadcast,
  router: Broadcast,
  switch: Broadcast,
  firewall: ShieldWarning,
  "cloud service": Browser,
  "cloud-service": Browser,
  "custom resource": SquaresFour,
  "custom-resource": SquaresFour,
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
  default: Pulse,
};

export function ResourceTypeIcon({
  type,
  className,
}: {
  type: string;
  className?: string;
}) {
  const Icon = RESOURCE_TYPE_ICONS[type.toLowerCase()] ?? Pulse;
  return <Icon className={cn("shrink-0", className)} aria-hidden />;
}

function statusTone(status: string): string {
  switch (status) {
    case "active":
    case "up":
    case "ok":
    case "healthy":
      return "border-transparent bg-success/12 text-success";
    case "warning":
    case "degraded":
      return "border-transparent bg-warning/15 text-warning";
    case "down":
    case "error":
    case "failed":
    case "critical":
      return "border-transparent bg-destructive/12 text-destructive";
    default:
      return "border-transparent bg-muted text-muted-foreground";
  }
}
