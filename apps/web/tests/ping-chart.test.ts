import { describe, expect, it } from "vitest";

import { toDownMarkArea } from "@/entities/resource/ui/monitoring/ping/PingMetricChart";
import type { DownInterval } from "@/entities/resource/ui/monitoring/ping/ping-metrics";

describe("toDownMarkArea", () => {
  it("maps down intervals to an echarts markArea data array", () => {
    const intervals: DownInterval[] = [
      { start: "2026-01-01T00:05:00Z", end: "2026-01-01T00:15:00Z" },
    ];
    expect(toDownMarkArea(intervals)).toEqual([
      {
        name: "Down",
        xAxis: ["2026-01-01T00:05:00Z", "2026-01-01T00:15:00Z"],
        itemStyle: { color: "rgba(220,48,53,0.08)" },
      },
    ]);
  });

  it("returns an empty array for no down intervals", () => {
    expect(toDownMarkArea([])).toEqual([]);
  });
});
