"use client";

import type { useTranslations } from "next-intl";

import { RelativeTime } from "@/shared/ui/relative-time";
import { AgentStatusBadge } from "@/features/probe-management/ui/probe-table";
import type { AgentColumn } from "@/features/probe-management/ui/probe-table";
import type { ProbeAgent } from "@/entities/agent/model/types";
import { cn } from "@/shared/utils/cn";

// A probe is "connected" when it is operational and its last heartbeat is
// recent. The lifecycle status badge still reflects the authoritative state.
export function isAgentConnected(agent: ProbeAgent): boolean {
  if (!agent.last_seen_at) return false;
  const last = new Date(agent.last_seen_at).getTime();
  if (Number.isNaN(last)) return false;
  return Date.now() - last < 3 * 60 * 1000;
}

function StatusCell({ agent }: { agent: ProbeAgent }) {
  const connected = isAgentConnected(agent);
  return (
    <div className="flex items-center gap-2">
      <span
        className={cn(
          "size-2 shrink-0 rounded-full",
          connected ? "bg-emerald-500" : "bg-muted-foreground/40",
        )}
        title={connected ? "Connected" : "Disconnected"}
      />
      <AgentStatusBadge status={agent.status} />
    </div>
  );
}

export function getAgentColumns(
  t: ReturnType<typeof useTranslations>,
): AgentColumn[] {
  return [
    {
      key: "name",
      header: t("name"),
      cell: (agent) => (
        <div className="min-w-0">
          <div className="max-w-[200px] truncate font-medium">{agent.name}</div>
          <div dir="ltr" className="max-w-[200px] truncate font-mono text-xs text-muted-foreground">
            {agent.hostname}
          </div>
        </div>
      ),
    },
    {
      key: "location",
      header: t("location"),
      cell: (agent) => (
        <span className="text-muted-foreground">
          {[agent.city, agent.country].filter(Boolean).join(", ") || "—"}
        </span>
      ),
    },
    {
      key: "publicIp",
      header: t("ip"),
      cell: (agent) => (
        <span dir="ltr" className="font-mono text-xs">
          {agent.public_ip || "—"}
        </span>
      ),
    },
    {
      key: "status",
      header: t("status"),
      cell: (agent) => <StatusCell agent={agent} />,
    },
    {
      key: "runningJobs",
      header: t("runningJobs"),
      cell: (agent) => (
        <span className="tabular-nums">{agent.running_jobs ?? 0}</span>
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
