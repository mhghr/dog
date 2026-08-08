"use client";

import { motion } from "motion/react";

interface ConnectionLineProps {
  source: { x: number; y: number };
  target: { x: number; y: number };
  status: "online" | "warning" | "offline";
  delay?: number;
}

const GRADIENT_IDS = {
  online: "conn-grad-online",
  warning: "conn-grad-warning",
  offline: "conn-grad-offline",
};

function createCurvedPath(
  start: { x: number; y: number },
  end: { x: number; y: number },
) {
  const midX = (start.x + end.x) / 2;
  const midY = Math.min(start.y, end.y) - 50;
  return `M ${start.x} ${start.y} Q ${midX} ${midY} ${end.x} ${end.y}`;
}

export function ConnectionLine({
  source,
  target,
  status,
  delay = 0,
}: ConnectionLineProps) {
  const gradientId = GRADIENT_IDS[status];

  return (
    <motion.path
      d={createCurvedPath(source, target)}
      fill="none"
      stroke={`url(#${gradientId})`}
      strokeWidth="1.2"
      initial={{ pathLength: 0, opacity: 0 }}
      animate={{ pathLength: 1, opacity: 0.7 }}
      transition={{
        duration: 1.2,
        delay,
        ease: "easeOut",
      }}
    />
  );
}

export function ConnectionGradients() {
  return (
    <defs>
      <linearGradient
        id="conn-grad-online"
        x1="0%"
        y1="0%"
        x2="100%"
        y2="0%"
      >
        <stop offset="0%" stopColor="#22c55e" stopOpacity="0" />
        <stop offset="10%" stopColor="#22c55e" stopOpacity="0.8" />
        <stop offset="90%" stopColor="#22c55e" stopOpacity="0.8" />
        <stop offset="100%" stopColor="#22c55e" stopOpacity="0" />
      </linearGradient>
      <linearGradient
        id="conn-grad-warning"
        x1="0%"
        y1="0%"
        x2="100%"
        y2="0%"
      >
        <stop offset="0%" stopColor="#eab308" stopOpacity="0" />
        <stop offset="10%" stopColor="#eab308" stopOpacity="0.8" />
        <stop offset="90%" stopColor="#eab308" stopOpacity="0.8" />
        <stop offset="100%" stopColor="#eab308" stopOpacity="0" />
      </linearGradient>
      <linearGradient
        id="conn-grad-offline"
        x1="0%"
        y1="0%"
        x2="100%"
        y2="0%"
      >
        <stop offset="0%" stopColor="#ef4444" stopOpacity="0" />
        <stop offset="10%" stopColor="#ef4444" stopOpacity="0.8" />
        <stop offset="90%" stopColor="#ef4444" stopOpacity="0.8" />
        <stop offset="100%" stopColor="#ef4444" stopOpacity="0" />
      </linearGradient>
    </defs>
  );
}
