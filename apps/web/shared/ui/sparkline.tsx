"use client";

import { useId } from "react";

// Lightweight SVG sparkline. No chart library — just a polyline rendered from
// the given values, cheap enough to render dozens of cards without jank.
export function Sparkline({
  values,
  width = 96,
  height = 28,
  stroke = "currentColor",
  className,
}: {
  values: number[];
  width?: number;
  height?: number;
  stroke?: string;
  className?: string;
}) {
  const gradientId = useId();

  if (values.length < 2) {
    return <span className={className} style={{ width, height }} aria-hidden />;
  }

  const min = Math.min(...values);
  const max = Math.max(...values);
  const range = max - min || 1;

  const points = values.map((v, i) => {
    const x = (i / (values.length - 1)) * width;
    const y = height - 2 - ((v - min) / range) * (height - 4);
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  });

  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      width={width}
      height={height}
      className={className}
      aria-hidden
      preserveAspectRatio="none"
    >
      <defs>
        <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor={stroke} stopOpacity="0.25" />
          <stop offset="100%" stopColor={stroke} stopOpacity="0" />
        </linearGradient>
      </defs>
      <polygon
        points={`0,${height} ${points.join(" ")} ${width},${height}`}
        fill={`url(#${gradientId})`}
      />
      <polyline
        points={points.join(" ")}
        fill="none"
        stroke={stroke}
        strokeWidth="1.5"
        strokeLinecap="round"
        strokeLinejoin="round"
        vectorEffect="non-scaling-stroke"
      />
    </svg>
  );
}
