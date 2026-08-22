import { describe, expect, it } from "vitest";

import { toDownMarkArea } from "./PingMetricChart";
import type { DownInterval } from "./ping-metrics";

describe("toDownMarkArea", () => {
  it("returns an empty array for no intervals", () => {
    expect(toDownMarkArea([], "#f00")).toEqual([]);
  });

  it("maps intervals to the echarts markArea shape", () => {
    const intervals: DownInterval[] = [
      { start: "2026-01-01T00:00:00Z", end: "2026-01-01T00:10:00Z" },
      { start: "2026-01-01T01:00:00Z", end: "2026-01-01T01:05:00Z" },
    ];
    const result = toDownMarkArea(intervals, "#ff3b30");

    expect(result).toHaveLength(2);
    expect(result[0].name).toBe("Down");
    expect(result[0].xAxis).toEqual([
      "2026-01-01T00:00:00Z",
      "2026-01-01T00:10:00Z",
    ]);
    expect(result[0].itemStyle.color).toContain("rgba(");
    expect(result[1].xAxis[1]).toBe("2026-01-01T01:05:00Z");
  });

  it("produces a translucent fill from the danger color", () => {
    const [area] = toDownMarkArea(
      [{ start: "2026-01-01T00:00:00Z", end: "2026-01-01T00:05:00Z" }],
      "#ff3b30",
    );
    expect(area.itemStyle.color).toMatch(/^rgba\(255,\s*59,\s*48,\s*0\.1\)$/);
  });
});
