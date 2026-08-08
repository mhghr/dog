import { describe, expect, it } from "vitest";

import {
  buildProbeConfig,
  createMonitorFormSchema,
  defaultFormValues,
  monitorToFormValues,
} from "@/features/monitor-management/schemas/schemas";
import type { Monitor } from "@/entities/monitor/model/types";

const t = (key: string) => key;
const schema = createMonitorFormSchema(t);

describe("monitorFormSchema", () => {
  it("accepts a valid http monitor", () => {
    const values = defaultFormValues("http");
    values.name = "Main Website";
    values.target = "https://example.com";

    const parsed = schema.safeParse(values);
    expect(parsed.success).toBe(true);
  });

  it("rejects http target without scheme", () => {
    const values = defaultFormValues("http");
    values.name = "Broken";
    values.target = "example.com";

    const parsed = schema.safeParse(values);
    expect(parsed.success).toBe(false);
  });

  it("requires a port for tcp monitors without target port", () => {
    const values = defaultFormValues("tcp");
    values.name = "Database";
    values.target = "db.example.com";
    values.interval_seconds = 60;

    const parsed = schema.safeParse(values);
    expect(parsed.success).toBe(false);
  });

  it("enforces per-type minimum interval", () => {
    const values = defaultFormValues("tls");
    values.name = "Cert";
    values.target = "example.com";
    values.interval_seconds = 30;

    const parsed = schema.safeParse(values);
    expect(parsed.success).toBe(false);
  });
});

describe("buildProbeConfig", () => {
  it("builds http config with parsed status codes and headers", () => {
    const values = defaultFormValues("http");
    values.http_expected_status_codes = "200, 204";
    values.http_headers = "X-Test: one\nAuthorization: Bearer token";

    const config = buildProbeConfig(values);

    expect(config.expected_status_codes).toEqual([200, 204]);
    expect(config.headers).toEqual({
      "X-Test": "one",
      Authorization: "Bearer token",
    });
    expect(config.method).toBe("GET");
  });

  it("builds ntp config with thresholds", () => {
    const values = defaultFormValues("ntp");

    const config = buildProbeConfig(values);

    expect(config.port).toBe(123);
    expect(config.version).toBe(4);
    expect(config.max_offset_millis).toBe(1000);
  });
});

describe("monitorToFormValues", () => {
  it("round-trips a stored monitor config", () => {
    const monitor: Monitor = {
      id: "id",
      name: "SMTP",
      type: "smtp",
      target: "mail.example.com",
      interval_seconds: 60,
      timeout_millis: 10000,
      retries: 1,
      enabled: true,
      config: {
        port: 587,
        mode: "starttls",
        require_starttls: true,
        expected_capabilities: ["STARTTLS", "SIZE"],
      },
      last_status: "up",
      last_checked_at: null,
      next_run_at: new Date().toISOString(),
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    };

    const values = monitorToFormValues(monitor);

    expect(values.smtp_port).toBe(587);
    expect(values.smtp_mode).toBe("starttls");
    expect(values.smtp_expected_capabilities).toBe("STARTTLS, SIZE");

    const rebuilt = buildProbeConfig(values);
    expect(rebuilt.port).toBe(587);
    expect(rebuilt.expected_capabilities).toEqual(["STARTTLS", "SIZE"]);
  });
});
