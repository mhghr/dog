"use client";

import { useMemo } from "react";
import { useQueries } from "@tanstack/react-query";

import { MonitorTypeIcon } from "./monitor-type-icon";
import { StatusBadge } from "@/design-system/components";
import {
  buildMetricsQueryString,
  resourceMonitorMetricsQueryKey,
  type MetricsRange,
} from "@/entities/resource/hooks/resource-query";
import { resourcesApi } from "@/entities/resource/api/resource.api";
import type { Monitor } from "@/entities/resource/hooks/types";
import type { MonitorTypeDef } from "@/entities/resource/model/types";
import type { ProbeResult } from "@/entities/monitor/model/result";
import type { ProbeSeries } from "@/entities/resource/api/resource.api";
import { toHttpProbeStats, summarizeHttp } from "../monitoring/http/http-metrics";
import { summarizeTcp } from "../monitoring/tcp/tcp-metrics";
import { summarizeDns } from "../monitoring/dns/dns-metrics";
import { summarizeTls } from "../monitoring/tls/tls-metrics";
import { summarize } from "../monitoring/ping/ping-metrics";

export type SummaryHealth = "healthy" | "warning" | "critical" | "unknown";

export interface MonitorSummaryCard {
  monitorId: string;
  title: string;
  type: string;
  value: string;
  unit?: string;
  sub: string;
  health: SummaryHealth;
  probes: number;
}

function healthOf(status: string | undefined): SummaryHealth {
  switch (status) {
    case "up":
      return "healthy";
    case "down":
      return "critical";
    case "paused":
      return "warning";
    default:
      return "unknown";
  }
}

function avg(latest: ProbeResult[], pick: (r: ProbeResult) => number | null): number | null {
  const values = latest.map(pick).filter((v): v is number => v != null);
  return values.length ? values.reduce((a, b) => a + b, 0) / values.length : null;
}

function deriveCard(
  monitor: Monitor,
  type: MonitorTypeDef | undefined,
  latest: ProbeResult[],
  series: ProbeSeries[],
): MonitorSummaryCard {
  const typeName = type?.name ?? monitor.name;
  const key = type?.executor_key ?? type?.slug ?? "";
  const probes = latest.length;
  const base: MonitorSummaryCard = {
    monitorId: monitor.id,
    title: typeName,
    type: key || "monitor",
    value: "—",
    sub: "—",
    health: healthOf(monitor.last_status),
    probes,
  };

  switch (key) {
    case "ping":
      return derivePingCard(base, latest);
    case "http":
      return deriveHttpCard(base, latest, series);
    case "tcp":
      return deriveTcpCard(base, latest);
    case "dns":
      return deriveDnsCard(base, latest);
    case "tls":
      return deriveTlsCard(base, latest);
    case "snmp":
      return deriveSnmpCard(base, latest);
    default:
      return deriveGenericCard(base, latest);
  }
}

function derivePingCard(base: MonitorSummaryCard, latest: ProbeResult[]): MonitorSummaryCard {
  const s = summarize(latest);
  return {
    ...base,
    type: "ping",
    value: s.latency == null ? "—" : String(Math.round(s.latency)),
    unit: "ms",
    sub: `Loss ${s.packetLoss == null ? "—" : `${s.packetLoss.toFixed(1)}%`}`,
  };
}

function deriveHttpCard(base: MonitorSummaryCard, latest: ProbeResult[], series: ProbeSeries[]): MonitorSummaryCard {
  const stats = toHttpProbeStats(latest);
  const s = summarizeHttp(latest, series as never);
  const code = stats.find((x) => x.statusCode != null)?.statusCode;
  const errorRate = s.availability != null ? 100 - s.availability : null;
  return {
    ...base,
    type: "http",
    value: code != null ? String(code) : "—",
    unit: code != null && code >= 200 && code < 300 ? "OK" : undefined,
    sub: [
      s.responseTimeMs != null ? `Response ${Math.round(s.responseTimeMs)} ms` : null,
      errorRate != null ? `Error ${errorRate.toFixed(1)}%` : null,
    ].filter(Boolean).join(" · ") || "—",
  };
}

function deriveTcpCard(base: MonitorSummaryCard, latest: ProbeResult[]): MonitorSummaryCard {
  const s = summarizeTcp(latest);
  return {
    ...base,
    type: "tcp",
    value: s.connectTimeMs == null ? "—" : String(Math.round(s.connectTimeMs)),
    unit: "ms",
    sub: "Connect time",
  };
}

function deriveDnsCard(base: MonitorSummaryCard, latest: ProbeResult[]): MonitorSummaryCard {
  const s = summarizeDns(latest);
  return {
    ...base,
    type: "dns",
    value: s.responseTimeMs == null ? "—" : String(Math.round(s.responseTimeMs)),
    unit: "ms",
    sub: `Answers ${s.answerCount == null ? "—" : s.answerCount}`,
  };
}

function deriveTlsCard(base: MonitorSummaryCard, latest: ProbeResult[]): MonitorSummaryCard {
  const s = summarizeTls(latest);
  return {
    ...base,
    type: "tls",
    value: s.certificateExpiryDays == null ? "—" : String(Math.round(s.certificateExpiryDays)),
    unit: "d",
    sub: s.handshakeTimeMs != null ? `Handshake ${Math.round(s.handshakeTimeMs)} ms` : "—",
  };
}

function deriveSnmpCard(base: MonitorSummaryCard, latest: ProbeResult[]): MonitorSummaryCard {
  const metrics = latest[0]?.metrics ?? {};
  const cpu = metrics["device.cpu_percent"];
  const mem = metrics["device.memory_percent"];
  return {
    ...base,
    type: "snmp",
    value: cpu != null && typeof cpu === "number" ? `${Math.round(cpu)}` : "—",
    unit: "%",
    sub: mem != null && typeof mem === "number" ? `Mem ${Math.round(mem)}%` : "CPU utilization",
  };
}

function deriveGenericCard(base: MonitorSummaryCard, latest: ProbeResult[]): MonitorSummaryCard {
  const s = avg(latest, (r) => r.duration_millis);
  return {
    ...base,
    value: s == null ? "—" : String(Math.round(s)),
    unit: "ms",
    sub: "Duration",
  };
}

const SUMMARY_RANGE: MetricsRange = "1h";

// Fetches the summary signal for every enabled monitor. Shares the exact
// query keys the monitoring views use, so the data is served from the cache
// instead of issuing a second request per monitor.
export function useMonitorSummaryCards(
  resourceId: string | undefined,
  monitors: Monitor[],
  types: MonitorTypeDef[],
) {
  const queryResults = useQueries({
    queries: monitors.map((monitor) => ({
      queryKey: resourceMonitorMetricsQueryKey(resourceId, monitor.id, SUMMARY_RANGE),
      queryFn: () =>
        resourcesApi.getMonitorMetrics(
          resourceId!,
          monitor.id,
          buildMetricsQueryString(SUMMARY_RANGE),
        ),
      enabled: Boolean(resourceId && monitor.id),
      staleTime: 15_000,
      refetchInterval: 60_000,
    })),
  });

  const cards = useMemo(() => {
    return monitors.map((monitor, index) => {
      const type = types.find((t) => t.id === monitor.monitor_type_id);
      const data = queryResults[index]?.data;
      return deriveCard(monitor, type, data?.latest ?? [], data?.series ?? []);
    });
  }, [monitors, types, queryResults]);

  return { cards, isPending: queryResults.some((r) => r?.isPending) };
}

interface ResourceSummaryProps {
  cards: MonitorSummaryCard[];
  isFa: boolean;
  activeId: string | null;
  onSelect: (monitorId: string) => void;
  isPending?: boolean;
}

// Sticky summary row with one card per enabled monitoring type. Clicking a
// card scrolls to that monitor's section; the card stays active.
export function ResourceSummary({
  cards,
  isFa,
  activeId,
  onSelect,
  isPending,
}: ResourceSummaryProps) {
  if (cards.length === 0) return null;

  return (
    <div className="sticky top-3 z-20 flex flex-wrap gap-2.5 rounded-2xl border border-border/50 bg-card/80 px-4 py-3 shadow-subtle backdrop-blur">
      {cards.map((card) => (
        <SummaryCardButton
          key={card.monitorId}
          card={card}
          isFa={isFa}
          active={activeId === card.monitorId}
          onSelect={onSelect}
        />
      ))}
    </div>
  );
}

function cardIconClass(health: SummaryHealth): string {
  if (health === "critical") return "bg-destructive/10 text-destructive";
  if (health === "warning") return "bg-warning/10 text-warning";
  return "bg-primary/10";
}

function cardTone(health: SummaryHealth): "success" | "warning" | "destructive" | "muted" {
  if (health === "healthy") return "success";
  if (health === "warning") return "warning";
  if (health === "critical") return "destructive";
  return "muted";
}

function cardLabel(health: SummaryHealth, isFa: boolean): string {
  if (isFa) {
    if (health === "healthy") return "سالم";
    if (health === "warning") return "هشدار";
    if (health === "critical") return "بحرانی";
    return "نامشخص";
  }
  return health;
}

function SummaryCardButton({
  card,
  isFa,
  active,
  onSelect,
}: {
  card: MonitorSummaryCard;
  isFa: boolean;
  active: boolean;
  onSelect: (monitorId: string) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onSelect(card.monitorId)}
      className={[
        "group flex min-w-40 flex-1 items-center gap-3 rounded-xl border px-3.5 py-2.5 text-start transition-all duration-200",
        active
          ? "border-primary/70 bg-primary/[0.06] shadow-[0_0_16px_-6px_var(--primary)]"
          : "border-border/60 bg-card/60 hover:border-primary/40 hover:bg-card",
      ].join(" ")}
    >
      <span className={`grid size-9 shrink-0 place-items-center rounded-lg text-primary ${cardIconClass(card.health)}`}>
        <MonitorTypeIcon type={card.type} className="size-4" />
      </span>

      <span className="min-w-0 flex-1">
        <span className="flex items-center justify-between gap-2">
          <span className="truncate text-xs font-semibold text-foreground">{card.title}</span>
          <StatusBadge tone={cardTone(card.health)} label={cardLabel(card.health, isFa)} />
        </span>
        <span className="mt-1 flex items-end justify-between gap-2">
          <span className="text-lg font-bold leading-none tabular-nums text-foreground" dir="ltr">
            {card.value}
            {card.unit && (
              <span className="ms-1 text-[11px] font-medium text-muted-foreground">{card.unit}</span>
            )}
          </span>
          <span className="text-[10px] text-muted-foreground">
            {isFa ? `${card.probes} پراب` : `${card.probes} probes`}
          </span>
        </span>
        <span className="mt-1 block truncate text-[10px] text-muted-foreground" dir="auto">
          {card.sub}
        </span>
      </span>
    </button>
  );
}
