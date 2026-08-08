"use client";

import { useMemo } from "react";
import DottedMap from "dotted-map";
import { useTheme } from "next-themes";
import { useMonitoring } from "@/entities/probe/hooks/use-monitoring";
import { ProbeMarker } from "./probe-marker";
import { UserMarker } from "./user-marker";
import { ConnectionLine, ConnectionGradients } from "./connection-line";
import { cn } from "@/shared/utils/cn";

/**
 * Converts geographic coordinates to SVG pixel coordinates.
 * Map bounds: longitude [-180, 180] maps to x [0, 800], latitude [90, -90] maps to y [0, 400].
 */
function projectPoint(lat: number, lng: number) {
  const x = (lng + 180) * (800 / 360);
  const y = (90 - lat) * (400 / 180);
  return { x, y };
}

export function WorldMonitoringMap({ className }: { className?: string }) {
  const { probes, userNodes, connections } = useMonitoring();
  const { resolvedTheme } = useTheme();
  const isDark = resolvedTheme === "dark";
  const bgColor = isDark ? "#060B14" : "#ffffff";

  const map = useMemo(
    () => new DottedMap({ height: 100, grid: "diagonal" }),
    [],
  );

  const svgMap = useMemo(
    () =>
      map.getSVG({
        radius: 0.22,
        color: isDark ? "#FFFFFF40" : "#00000040",
        shape: "circle",
        backgroundColor: bgColor,
      }),
    [map, isDark, bgColor],
  );

  return (
    <div
      className={cn(
        "relative w-full overflow-hidden rounded-xl bg-background",
        className,
      )}
    >
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src={`data:image/svg+xml;utf8,${encodeURIComponent(svgMap)}`}
        className="pointer-events-none h-full w-full select-none object-cover [mask-image:linear-gradient(to_bottom,transparent,white_10%,white_90%,transparent)]"
        alt="world map"
        draggable={false}
      />
      <svg
        viewBox="0 0 800 400"
        className="pointer-events-none absolute inset-0 h-full w-full select-none"
      >
        <ConnectionGradients />

        {connections.map((conn, i) => {
          const start = projectPoint(conn.source.lat, conn.source.lng);
          const end = projectPoint(conn.target.lat, conn.target.lng);
          return (
            <ConnectionLine
              key={conn.id}
              source={start}
              target={end}
              status={conn.status}
              delay={0.3 * i}
            />
          );
        })}

        <g className="pointer-events-auto">
          {userNodes.map((node) => {
            const p = projectPoint(node.latitude, node.longitude);
            return <UserMarker key={node.id} user={node} x={p.x} y={p.y} />;
          })}
          {probes.map((probe) => {
            const p = projectPoint(probe.latitude, probe.longitude);
            return <ProbeMarker key={probe.id} probe={probe} x={p.x} y={p.y} />;
          })}
        </g>
      </svg>
    </div>
  );
}
