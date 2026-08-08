"use client";

import type { Probe } from "@/entities/probe/model/types";

interface MonitoringTooltipProps {
  probe: Probe;
}

export function MonitoringTooltip({ probe }: MonitoringTooltipProps) {
  return (
    <div className="rounded-lg border border-border bg-card/95 p-2.5 text-xs shadow-lg backdrop-blur">
      <p className="font-semibold text-foreground">{probe.name}</p>
      <p className="mt-0.5 text-muted-foreground">
        {probe.city}, {probe.country}
      </p>
      <div className="mt-2 flex flex-col gap-0.5">
        <div className="flex items-center justify-between gap-3">
          <span className="text-muted-foreground">Latency</span>
          <span className="font-mono tabular-nums text-foreground">
            {probe.latency}ms
          </span>
        </div>
        <div className="flex items-center justify-between gap-3">
          <span className="text-muted-foreground">Packet Loss</span>
          <span className="font-mono tabular-nums text-foreground">
            {probe.packetLoss}%
          </span>
        </div>
        <div className="flex items-center justify-between gap-3">
          <span className="text-muted-foreground">Status</span>
          <span
            className="inline-flex items-center gap-1.5 font-medium capitalize"
            style={{
              color:
                probe.status === "online"
                  ? "#22c55e"
                  : probe.status === "warning"
                    ? "#eab308"
                    : "#ef4444",
            }}
          >
            <span
              className="size-1.5 rounded-full"
              style={{ backgroundColor: "currentColor" }}
            />
            {probe.status}
          </span>
        </div>
      </div>
    </div>
  );
}
