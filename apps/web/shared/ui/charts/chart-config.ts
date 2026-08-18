import type { EChartsCoreOption } from "echarts/core";

export interface ChartPalette {
  text: string;
  axis: string;
  primary: string;
  success: string;
  warning: string;
  danger: string;
  tooltipBg: string;
  tooltipText: string;
  series: string[];
}

export function makeGrid(
  overrides?: Partial<EChartsCoreOption["grid"]>,
): NonNullable<EChartsCoreOption["grid"]> {
  return { top: 16, right: 16, bottom: 32, left: 56, ...overrides };
}

export function makeTimeXAxis(
  locale: string,
  palette: ChartPalette,
  fontFamily = "inherit",
): NonNullable<EChartsCoreOption["xAxis"]> {
  return {
    type: "time" as const,
    axisLine: { show: false },
    axisTick: { show: false },
    axisLabel: {
      color: palette.text,
      fontFamily,
      hideOverlap: true,
      formatter: (value: number) =>
        new Intl.DateTimeFormat(locale, {
          hour: "2-digit",
          minute: "2-digit",
        }).format(new Date(value)),
    },
    splitLine: { show: false },
  };
}

export function makeTooltip(
  palette: ChartPalette,
  valueFormatter: (value: unknown) => string,
): NonNullable<EChartsCoreOption["tooltip"]> {
  return {
    trigger: "axis" as const,
    backgroundColor: palette.tooltipBg,
    borderColor: palette.axis,
    textStyle: { color: palette.tooltipText, fontSize: 12 },
    valueFormatter,
  };
}

// Converts a #RRGGBB hex color to an rgba() string with the given alpha.
// Used to derive translucent fills (mark areas, chart gradients) from palette
// colors while staying theme-aware.
export function hexToRgba(hex: string, alpha: number): string {
  const clean = hex.replace("#", "").trim();
  if (/^[0-9a-fA-F]{3}$/.test(clean) || /^[0-9a-fA-F]{6}$/.test(clean)) {
    const full =
      clean.length === 3 ? clean.split("").map((c) => c + c).join("") : clean;
    const r = parseInt(full.slice(0, 2), 16);
    const g = parseInt(full.slice(2, 4), 16);
    const b = parseInt(full.slice(4, 6), 16);
    if ([r, g, b].every((n) => !Number.isNaN(n))) {
      return `rgba(${r},${g},${b},${alpha})`;
    }
  }

  // Non-hex CSS colors (oklch/hsl/rgb/... ) are resolved through a 1px canvas
  // so theme tokens like `--chart-1: oklch(...)` still produce translucent
  // fills. Falls back to the raw color outside a browser (SSR).
  if (typeof document !== "undefined") {
    try {
      const canvas = document.createElement("canvas");
      canvas.width = canvas.height = 1;
      const ctx = canvas.getContext("2d");
      if (ctx) {
        ctx.clearRect(0, 0, 1, 1);
        ctx.fillStyle = hex;
        ctx.globalAlpha = alpha;
        ctx.fillRect(0, 0, 1, 1);
        const [r, g, b, a] = ctx.getImageData(0, 0, 1, 1).data;
        return `rgba(${r},${g},${b},${(a / 255).toFixed(3)})`;
      }
    } catch {
      // fall through to the raw color
    }
  }
  return hex;
}
