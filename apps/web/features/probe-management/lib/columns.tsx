import type { useTranslations } from "next-intl";

import { RelativeTime } from "@/shared/ui/relative-time";
import { AgentStatusBadge } from "@/features/probe-management/ui/probe-table";
import type { AgentColumn } from "@/features/probe-management/ui/probe-table";
import type { ProbeAgent } from "@/entities/agent/model/types";

export function getAgentColumns(
  t: ReturnType<typeof useTranslations>,
): AgentColumn[] {
  return [
    {
      key: "name",
      header: t("name"),
      cell: (agent) => (
        <span className="max-w-[180px] truncate font-medium">{agent.name}</span>
      ),
    },
    {
      key: "hostname",
      header: t("hostname"),
      cell: (agent) => (
        <span dir="ltr" className="font-mono text-xs">
          {agent.hostname}
        </span>
      ),
    },
    {
      key: "location",
      header: t("location"),
      cell: (agent) => (
        <span className="text-muted-foreground">{agent.city ?? "—"}</span>
      ),
    },
    {
      key: "version",
      header: t("version"),
      cell: (agent) => (
        <span dir="ltr" className="font-mono text-xs">
          {agent.version || "—"}
        </span>
      ),
    },
    {
      key: "status",
      header: t("status"),
      cell: (agent) => <AgentStatusBadge status={agent.status} />,
    },
    {
      key: "lastSeen",
      header: t("lastSeen"),
      cell: (agent: ProbeAgent) => (
        <span className="text-sm text-muted-foreground">
          {agent.last_seen_at ? <RelativeTime value={agent.last_seen_at} /> : "—"}
        </span>
      ),
    },
  ];
}
