import { describe, expect, it } from "vitest";

import {
  formatDuration,
  formatInterval,
  formatPercent,
  formatRelativeTime,
} from "@/lib/formatters";

describe("formatDuration", () => {
  it("formats milliseconds below one second", () => {
    expect(formatDuration(184, "en")).toBe("184 ms");
  });

  it("formats seconds above one second", () => {
    expect(formatDuration(5250, "en")).toBe("5.25 s");
  });

  it("handles missing values", () => {
    expect(formatDuration(null, "en")).toBe("—");
    expect(formatDuration(undefined, "en")).toBe("—");
  });
});

describe("formatPercent", () => {
  it("formats percentages", () => {
    expect(formatPercent(99.95, "en")).toBe("99.95%");
  });

  it("handles null", () => {
    expect(formatPercent(null, "en")).toBe("—");
  });
});

describe("formatInterval", () => {
  it("uses compact units", () => {
    expect(formatInterval(60, "en")).toBe("1m");
    expect(formatInterval(3600, "en")).toBe("1h");
    expect(formatInterval(43200, "en")).toBe("12h");
    expect(formatInterval(86400, "en")).toBe("1d");
    expect(formatInterval(45, "en")).toBe("45s");
  });
});

describe("formatRelativeTime", () => {
  it("returns placeholder for invalid input", () => {
    expect(formatRelativeTime(null, "en")).toBe("—");
    expect(formatRelativeTime("not-a-date", "en")).toBe("—");
  });

  it("formats past timestamps", () => {
    const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000).toISOString();
    expect(formatRelativeTime(fiveMinutesAgo, "en")).toContain("minute");
  });
});
