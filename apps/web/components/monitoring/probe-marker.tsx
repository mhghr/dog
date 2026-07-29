"use client";

import { useState } from "react";
import type { Probe } from "@/types/monitoring";
import { MonitoringTooltip } from "./monitoring-tooltip";

const STATUS_COLORS: Record<Probe["status"], string> = {
  online: "#22c55e",
  warning: "#eab308",
  offline: "#ef4444",
};

interface ProbeMarkerProps {
  probe: Probe;
  x: number;
  y: number;
}

export function ProbeMarker({ probe, x, y }: ProbeMarkerProps) {
  const [hovered, setHovered] = useState(false);
  const color = STATUS_COLORS[probe.status];

  return (
    <g
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{ cursor: "pointer" }}
    >
      {/* Label */}
      <text
        x={x}
        y={y - 9}
        textAnchor="middle"
        fill="var(--foreground)"
        className="text-[8px] font-display font-medium"
      >
        {probe.city}
      </text>
      <circle cx={x} cy={y} r="3" fill={color} />
      <circle cx={x} cy={y} r="3" fill={color} opacity="0.4">
        <animate
          attributeName="r"
          from="3"
          to="10"
          dur="2s"
          begin="0s"
          repeatCount="indefinite"
        />
        <animate
          attributeName="opacity"
          from="0.4"
          to="0"
          dur="2s"
          begin="0s"
          repeatCount="indefinite"
        />
      </circle>
      {hovered && (
        <foreignObject x={x + 10} y={y - 60} width="180" height="100">
          <MonitoringTooltip probe={probe} />
        </foreignObject>
      )}
    </g>
  );
}
