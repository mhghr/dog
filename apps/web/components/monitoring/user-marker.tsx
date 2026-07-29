"use client";

import { useState } from "react";
import type { UserNode } from "@/types/monitoring";

interface UserMarkerProps {
  user: UserNode;
  x: number;
  y: number;
}

export function UserMarker({ user, x, y }: UserMarkerProps) {
  const [hovered, setHovered] = useState(false);

  return (
    <g
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{ cursor: "pointer" }}
    >
      {/* Pulse ring */}
      <circle cx={x} cy={y} r="8" fill="#6366f1" opacity="0.25">
        <animate
          attributeName="r"
          from="8"
          to="18"
          dur="1.5s"
          begin="0s"
          repeatCount="indefinite"
        />
        <animate
          attributeName="opacity"
          from="0.35"
          to="0"
          dur="1.5s"
          begin="0s"
          repeatCount="indefinite"
        />
      </circle>
      {/* Background circle */}
      <circle cx={x} cy={y} r="8" fill="#6366f1" />
      {/* Label */}
      <text
        x={x}
        y={y - 13}
        textAnchor="middle"
        fill="var(--foreground)"
        className="text-[8px] font-display font-medium"
      >
        {user.city}
      </text>
      {/* Monitor/server icon */}
      <g
        transform={`translate(${x - 7}, ${y - 6.5})`}
        fill="none"
        stroke="white"
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <rect x="0.5" y="0.5" width="13" height="9" rx="1.5" />
        <line x1="7" y1="9.5" x2="7" y2="12" />
        <line x1="3" y1="12" x2="11" y2="12" />
      </g>
      {hovered && (
        <foreignObject x={x + 12} y={y - 55} width="170" height="60">
          <div className="rounded-lg border border-border bg-card/95 p-2.5 text-xs shadow-lg backdrop-blur">
            <p className="font-semibold text-foreground">
              {user.city}, {user.country}
            </p>
            <p className="mt-0.5 font-mono text-muted-foreground">{user.ip}</p>
            <p className="text-muted-foreground">{user.isp}</p>
          </div>
        </foreignObject>
      )}
    </g>
  );
}
