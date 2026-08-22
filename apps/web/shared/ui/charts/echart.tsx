"use client";

import { useEffect, useMemo, useRef } from "react";
import { LineChart, PieChart } from "echarts/charts";
import {
  DataZoomComponent,
  GridComponent,
  LegendComponent,
  TooltipComponent,
} from "echarts/components";
import * as echarts from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import type { EChartsCoreOption } from "echarts/core";
import { useTheme } from "next-themes";

echarts.use([LineChart, PieChart, GridComponent, TooltipComponent, LegendComponent, DataZoomComponent, CanvasRenderer]);

function readToken(variableName: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(variableName).trim();
}

const FALLBACK_PALETTE = {
  text: "#6B6E8C",
  axis: "#D8DAE8",
  primary: "#3072F4",
  success: "#10B981",
  warning: "#F59E0B",
  danger: "#EF4444",
  tooltipBg: "#FFFFFF",
  tooltipText: "#11121C",
  series: ["#3072F4", "#0EA5E9", "#8B5CF6", "#10B981", "#F59E0B"],
};

export function useChartPalette() {
  useTheme();

  if (typeof document === "undefined") return FALLBACK_PALETTE;

  const read = (name: string, fallback: string) => {
    const value = readToken(name);
    return value || fallback;
  };

  return {
    text: read("--muted-foreground", FALLBACK_PALETTE.text),
    axis: read("--border", FALLBACK_PALETTE.axis),
    primary: read("--chart-1", FALLBACK_PALETTE.primary),
    success: read("--success", FALLBACK_PALETTE.success),
    warning: read("--warning", FALLBACK_PALETTE.warning),
    danger: read("--destructive", FALLBACK_PALETTE.danger),
    tooltipBg: read("--card", FALLBACK_PALETTE.tooltipBg),
    tooltipText: read("--card-foreground", FALLBACK_PALETTE.tooltipText),
    series: ["--chart-1", "--chart-2", "--chart-3", "--chart-4", "--chart-5"].map(
      (name, index) => read(name, FALLBACK_PALETTE.series[index]!),
    ),
  };
}

interface EChartProps {
  option: EChartsCoreOption;
  className?: string;
  ariaLabel?: string;
  /** ECharts instance events (click, dataZoom, ...) keyed by event name. */
  onEvents?: Record<string, (params: unknown) => void>;
}

export function EChart({ option, className, ariaLabel, onEvents }: EChartProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<echarts.ECharts | null>(null);

  const serializedOption = useMemo(() => option, [option]);

  useEffect(() => {
    if (!containerRef.current) {
      return;
    }

    const chart = echarts.init(containerRef.current);
    chartRef.current = chart;

    const observer = new ResizeObserver(() => chart.resize());
    observer.observe(containerRef.current);

    return () => {
      observer.disconnect();
      chart.dispose();
      chartRef.current = null;
    };
  }, []);

  useEffect(() => {
    if (!chartRef.current || !onEvents) return;
    const handlers = Object.entries(onEvents).map(([name, handler]) => {
      chartRef.current?.on(name, handler as (...args: unknown[]) => void);
      return { name, handler };
    });
    return () => {
      handlers.forEach(({ name, handler }) => {
        chartRef.current?.off(name, handler as (...args: unknown[]) => void);
      });
    };
  }, [onEvents]);

  useEffect(() => {
    chartRef.current?.setOption(serializedOption, { notMerge: true, lazyUpdate: true });
  }, [serializedOption]);

  return (
    <div
      ref={containerRef}
      role="img"
      aria-label={ariaLabel}
      className={className ?? "h-72 w-full"}
    />
  );
}
