"use client";

import { useEffect, useMemo, useRef } from "react";
import { LineChart } from "echarts/charts";
import {
  GridComponent,
  LegendComponent,
  TooltipComponent,
} from "echarts/components";
import * as echarts from "echarts/core";
import { CanvasRenderer } from "echarts/renderers";
import type { EChartsCoreOption } from "echarts/core";
import { useTheme } from "next-themes";

echarts.use([LineChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer]);

function readToken(variableName: string): string {
  return getComputedStyle(document.documentElement).getPropertyValue(variableName).trim();
}

const FALLBACK_PALETTE = {
  text: "#7A82A0",
  axis: "#E2E4EC",
  primary: "#4F66F0",
  success: "#0D9464",
  danger: "#DC3035",
  tooltipBg: "#FFFFFF",
  tooltipText: "#151829",
};

export function useChartPalette() {
  useTheme();

  if (typeof document === "undefined") return FALLBACK_PALETTE;

  return {
    text: readToken("--muted-foreground"),
    axis: readToken("--border"),
    primary: readToken("--chart-1"),
    success: readToken("--success"),
    danger: readToken("--destructive"),
    tooltipBg: readToken("--card"),
    tooltipText: readToken("--card-foreground"),
  };
}

interface EChartProps {
  option: EChartsCoreOption;
  className?: string;
  ariaLabel?: string;
}

export function EChart({ option, className, ariaLabel }: EChartProps) {
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
    chartRef.current?.setOption(serializedOption, { notMerge: true });
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
