"use client";

import { useId } from "react";

export interface SparklineSeries {
  name?: string;
  points: Array<{ time: string; value: number }>;
}

interface SparklineProps {
  series: SparklineSeries[];
  colors: string[];
  height?: number;
  lineWidth?: number;
  /** Opacity of the gradient fill right under the line (fades to 0). */
  fillOpacity?: number;
  className?: string;
  ariaLabel?: string;
}

// Lightweight multi-line sparkline (pure SVG, no axes/tooltips) for compact
// stat panels. `preserveAspectRatio="none"` + `vector-effect` keep the stroke
// crisp regardless of the container width, and each line gets a soft gradient
// fill that fades from the line color to transparent.
export function Sparkline({
  series,
  colors,
  height = 44,
  lineWidth = 1.5,
  fillOpacity = 0.22,
  className,
  ariaLabel,
}: SparklineProps) {
  const uid = useId().replace(/:/g, "");

  const visible = series.filter((s) => s.points.length > 0);

  if (visible.length === 0) {
    return <div role="img" aria-label={ariaLabel} className={className} style={{ height }} />;
  }

  const values = visible.flatMap((s) => s.points.map((p) => p.value));
  const min = Math.min(...values);
  const max = Math.max(...values);
  const span = max - min;
  const pad = span === 0 ? (Math.abs(max) * 0.1 || 1) : span * 0.12;
  const lo = min - pad;
  const hi = max + pad;

  const x = (index: number, len: number) => (len <= 1 ? 0 : (index / (len - 1)) * 100);
  const y = (value: number) => height - ((value - lo) / (hi - lo)) * height;

  const baseline = height;

  const rendered = visible.map((s, i) => {
    const color = colors[i % colors.length];
    const id = `spark-${uid}-${i}`;
    const n = s.points.length;
    const line =
      n === 1
        ? `0,${y(s.points[0]!.value)} 100,${y(s.points[0]!.value)}`
        : s.points.map((p, j) => `${x(j, n)},${y(p.value)}`).join(" ");
    const fillPath = `${line} ${100},${baseline} 0,${baseline}`;

    return { name: s.name ?? `series-${i}`, id, color, line, fillPath };
  });

  return (
    <svg
      viewBox={`0 0 100 ${height}`}
      preserveAspectRatio="none"
      role="img"
      aria-label={ariaLabel}
      className={className}
      style={{ display: "block", width: "100%", height }}
    >
      <defs>
        {rendered.map((r) => (
          <linearGradient key={r.id} id={r.id} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={r.color} stopOpacity={fillOpacity} />
            <stop offset="100%" stopColor={r.color} stopOpacity={0} />
          </linearGradient>
        ))}
      </defs>
      {rendered.map((r) => (
        <g key={r.name}>
          <polygon points={r.fillPath} fill={`url(#${r.id})`} />
          <polyline
            points={r.line}
            fill="none"
            stroke={r.color}
            strokeWidth={lineWidth}
            strokeLinejoin="round"
            strokeLinecap="round"
            vectorEffect="non-scaling-stroke"
          />
        </g>
      ))}
    </svg>
  );
}
