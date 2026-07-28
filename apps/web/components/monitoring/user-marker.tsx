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
      <circle cx={x} cy={y} r="3" fill="#3b82f6" />
      <circle cx={x} cy={y} r="3" fill="#3b82f6" opacity="0.4">
        <animate
          attributeName="r"
          from="3"
          to="12"
          dur="1.5s"
          begin="0s"
          repeatCount="indefinite"
        />
        <animate
          attributeName="opacity"
          from="0.5"
          to="0"
          dur="1.5s"
          begin="0s"
          repeatCount="indefinite"
        />
      </circle>
      {hovered && (
        <foreignObject x={x + 10} y={y - 55} width="170" height="60">
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
