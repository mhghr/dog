import { describe, expect, it } from "vitest";

import { readPingConfig, readPingThresholds } from "@/entities/resource/ui/monitoring/ping/ping-config";

describe("readPingThresholds", () => {
  it("returns defaults when configuration is empty", () => {
    const t = readPingThresholds(undefined);
    expect(t.latency).toEqual({ warning: 200, critical: 500 });
    expect(t.packetLoss).toEqual({ warning: 5, critical: 20 });
    expect(t.jitter).toEqual({ warning: 30, critical:80 });
  });

  it("reads thresholds from configuration.health_rules.latency_ms", () => {
    const t = readPingThresholds({
      health_rules: { latency_ms: { warning: 100, critical: 250 } },
    });
    expect(t.latency).toEqual({ warning: 100, critical: 250 });
  });

  it("accepts rtt_ms as a latency alias", () => {
    const t = readPingThresholds({
      health_rules: { rtt_ms: { warning: 120, critical: 300 } },
    });
    expect(t.latency).toEqual({ warning: 120, critical: 300 });
  });

  it("accepts packet_loss_percent as a packet-loss alias", () => {
    const t = readPingThresholds({
      health_rules: { packet_loss_percent: { warning: 2, critical: 8 } },
    });
    expect(t.packetLoss).toEqual({ warning: 2, critical: 8 });
  });

  it("falls back to default when a rule only has one threshold", () => {
    const t = readPingThresholds({
      health_rules: { jitter_ms: { warning: 10 } },
    });
    expect(t.jitter).toEqual({ warning: 10, critical: 80 });
  });
});

describe("readPingConfig", () => {
  it("reads packet_count and packet_interval_millis", () => {
    const cfg = readPingConfig({ packet_count: 8, packet_interval_millis: 500 });
    expect(cfg.packetCount).toBe(8);
    expect(cfg.packetIntervalMillis).toBe(500);
  });

  it("falls back to executor defaults", () => {
    const cfg = readPingConfig(undefined);
    expect(cfg.packetCount).toBe(4);
    expect(cfg.packetIntervalMillis).toBe(200);
  });

  it("accepts count as a packet_count alias", () => {
    const cfg = readPingConfig({ count: 3 });
    expect(cfg.packetCount).toBe(3);
  });
});
