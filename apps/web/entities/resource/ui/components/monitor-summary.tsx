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
import { toTcpProbeStats, summarizeTcp } from "../monitoring/tcp/tcp-metrics";
import { toDnsProbeStats, summarizeDns } from "../monitoring/dns/dns-metrics";
import { toTlsProbeStats, summarizeTls } from "../monitoring/tls/tls-metrics";
import { toProbeStats, summarize } from "../monitoring/ping/ping-metrics";

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
  const health = healthOf(monitor.last_status);
  const probes = latest.length;

  switch (key) {
    case "ping": {
      const s = summarize(latest);
      return {
        monitorId: monitor.id,
        title: typeName,
        type: "ping",
        value: s.latency == null ? "—" : String(Math.round(s.latency)),
        unit: "ms",
        sub: `Loss ${s.packetLoss == null ? "—" : `${s.packetLoss.toFixed(1)}%`}`,
        health,
        probes,
      };
    }
    case "http": {
      const stats = toHttpProbeStats(latest);
      const s = summarizeHttp(latest, series as never);
      const first = stats.find((x) => x.statusCode != null);
      const code = first?.statusCode;
      const errorRate = s.availability != null ? 100 - s.availability : null;
      return {
        monitorId: monitor.id,
        title: typeName,
        type: "http",
        value: code != null ? String(code) : "—",
        unit: code != null && code >= 200 && code < 300 ? "OK" : undefined,
        sub: [
          s.responseTimeMs != null ? `Response ${Math.round(s.responseTimeMs)} ms` : null,
          errorRate != null ? `Error ${errorRate.toFixed(1)}%` : null,
        ].filter(Boolean).join(" · ") || "—",
        health,
        probes,
      };
    }
    case "tcp": {
      const s = summarizeTcp(latest);
      return {
        monitorId: monitor.id,
        title: typeName,
        type: "tcp",
        value: s.connectTimeMs == null ? "—" : String(Math.round(s.connectTimeMs)),
        unit: "ms",
        sub: "Connect time",
        health,
        probes,
      };
    }
    case "dns": {
      const s = summarizeDns(latest);
      return {
        monitorId: monitor.id,
        title: typeName,
        type: "dns",
        value: s.responseTimeMs == null ? "—" : String(Math.round(s.responseTimeMs)),
        unit: "ms",
        sub: `Answers ${s.answerCount == null ? "—" : s.answerCount}`,
        health,
        probes,
      };
    }
    case "tls": {
      const s = summarizeTls(latest);
      return {
        monitorId: monitor.id,
        title: typeName,
        type: "tls",
        value: s.certificateExpiryDays == null ? "—" : String(Math.round(s.certificateExpiryDays)),
        unit: "d",
        sub: s.handshakeTimeMs != null ? `Handshake ${Math.round(s.handshakeTimeMs)} ms` : "—",
        health,
        probes,
      };
    }
    case "snmp": {
      const first = latest[0];
      const metrics = first?.metrics ?? {};
      const cpu = metrics["device.cpu_percent"];
      const mem = metrics["device.memory_percent"];
      return {
        monitorId: monitor.id,
        title: typeName,
        type: "snmp",
        value: cpu != null && typeof cpu === "number" ? `${Math.round(cpu)}` : "—",
        unit: "%",
        sub: mem != null && typeof mem === "number" ? `Mem ${Math.round(mem)}%` : "CPU utilization",
        health,
        probes,
      };
    }
    default: {
      // Generic fallback: average total duration across probes.
      const s = avg(latest, (r) => r.duration_millis);
      return {
        monitorId: monitor.id,
        title: typeName,
        type: key || "monitor",
        value: s == null ? "—" : String(Math.round(s)),
        unit: "ms",
        sub: "Duration",
        health,
        probes,
      };
    }
  }
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
      {cards.map((card) => {
        const active = activeId === card.monitorId;
        return (
          <button
            key={card.monitorId}
            type="button"
            onClick={() => onSelect(card.monitorId)}
            className={[
              "group flex min-w-40 flex-1 items-center gap-3 rounded-xl border px-3.5 py-2.5 text-start transition-all duration-200",
              active
                ? "border-primary/70 bg-primary/[0.06] shadow-[0_0_16px_-6px_var(--primary)]"
                : "border-border/60 bg-card/60 hover:border-primary/40 hover:bg-card",
            ].join(" ")}
          >
            <span
              className={`grid size-9 shrink-0 place-items-center rounded-lg text-primary ${
                card.health === "critical"
                  ? "bg-destructive/10 text-destructive"
                  : card.health === "warning"
                    ? "bg-warning/10 text-warning"
                    : "bg-primary/10"
              }`}
            >
              <MonitorTypeIcon type={card.type} className="size-4" />
            </span>

            <span className="min-w-0 flex-1">
              <span className="flex items-center justify-between gap-2">
                <span className="truncate text-xs font-semibold text-foreground">{card.title}</span>
                <StatusBadge
                  tone={
                    card.health === "healthy"
                      ? "success"
                      : card.health === "warning"
                        ? "warning"
                        : card.health === "critical"
                          ? "destructive"
                          : "muted"
                  }
                  label={isFa
                    ? card.health === "healthy" ? "سالم" : card.health === "warning" ? "هشدار" : card.health === "critical" ? "بحرانی" : "نامشخص"
                    : card.health}
                />
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
      })}
    </div>
  );
}
