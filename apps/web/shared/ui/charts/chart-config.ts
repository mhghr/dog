import type { EChartsCoreOption } from "echarts/core";

export interface ChartPalette {
  text: string;
  axis: string;
  primary: string;
  success: string;
  danger: string;
  tooltipBg: string;
  tooltipText: string;
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
