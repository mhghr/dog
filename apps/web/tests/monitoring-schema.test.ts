import { describe, expect, it } from "vitest";

import { getMonitoringSchema } from "@/entities/resource/ui/settings/schemas";
import type { MonitorTypeDef } from "@/entities/resource/model/types";

function typeDef(overrides: Partial<MonitorTypeDef>): MonitorTypeDef {
  return {
    id: "t1",
    name: "Test",
    slug: "test",
    category: "network",
    execution_type: "probe",
    executor_key: "test",
    description: "",
    icon: "activity",
    enabled: true,
    metric_keys: [],
    config_schema: {},
    default_configuration: {},
    metric_schema: {},
    health_parameters: {},
    supported_resource_types: [],
    created_at: "",
    updated_at: "",
    ...overrides,
  } as MonitorTypeDef;
}

describe("getMonitoringSchema", () => {
  it("returns the ping schema with configuration, execution and health rules", () => {
    const schema = getMonitoringSchema(typeDef({ executor_key: "ping" }));
    expect(schema.type).toBe("ping");
    expect(schema.configFields.length).toBeGreaterThan(0);
    expect(schema.healthRules.map((r) => r.key)).toEqual(
      expect.arrayContaining(["reachability", "latency_ms", "packet_loss", "jitter_ms"]),
    );
    // jitter is optional (disabled by default) but still present as a rule.
    const jitter = schema.healthRules.find((r) => r.key === "jitter_ms");
    expect(jitter?.direction).toBe("higher_is_worse");
    expect(jitter?.defaults).toEqual({ warning: 30, critical: 80 });
  });

  it("marks reachability rules as boolean", () => {
    for (const executorKey of ["ping", "http", "tcp", "dns", "tls"]) {
      const schema = getMonitoringSchema(typeDef({ executor_key: executorKey }));
      const availability = schema.healthRules.find((r) => r.key === "reachability");
      expect(availability?.direction).toBe("boolean");
    }
  });

  it("uses LOWER_IS_WORSE for certificate expiry", () => {
    const schema = getMonitoringSchema(typeDef({ executor_key: "tls" }));
    const expiry = schema.healthRules.find((r) => r.key === "certificate_expiry_days");
    expect(expiry?.direction).toBe("lower_is_worse");
    expect(expiry?.defaults).toEqual({ warning: 30, critical: 14 });
  });

  it("groups advanced fields into the advanced section", () => {
    const schema = getMonitoringSchema(typeDef({ executor_key: "http" }));
    const size = schema.configFields.find((f) => f.key === "max_response_size_bytes");
    expect(size?.section).toBe("advanced");
  });

  it("does not expose the IP version field in the forms", () => {
    for (const executorKey of ["ping", "http", "tcp", "dns", "tls"]) {
      const schema = getMonitoringSchema(typeDef({ executor_key: executorKey }));
      expect(schema.configFields.some((f) => f.key === "ip_version"), `${executorKey} should hide ip_version`).toBe(false);
    }
  });

  it("hides the HTTP request body behind body-bearing methods", () => {
    const schema = getMonitoringSchema(typeDef({ executor_key: "http" }));
    const body = schema.configFields.find((f) => f.key === "request_body");
    expect(body?.visibleWhen?.equalsAny).toContain("POST");
  });

  it("falls back to a generic schema for unknown types", () => {
    const schema = getMonitoringSchema(
      typeDef({
        executor_key: "smtp",
        name: "SMTP Service",
        config_schema: {
          type: "object",
          properties: {
            host: { type: "string" },
            port: { type: "integer", default: 587 },
            verify_tls: { type: "boolean", default: true },
          },
        },
        health_parameters: {
          reachability: { warning_threshold: 0, critical_threshold: 0 },
          response_time_ms: { warning_threshold: 500, critical_threshold: 2000 },
        },
      }),
    );
    expect(schema.type).toBe("smtp");
    expect(schema.configFields.map((f) => f.key)).toEqual(["host", "port", "verify_tls"]);
    expect(schema.configFields.find((f) => f.key === "verify_tls")?.widget).toBe("switch");
    expect(schema.configFields.find((f) => f.key === "port")?.widget).toBe("number");
    expect(schema.healthRules.map((r) => r.key)).toEqual(["reachability", "response_time_ms"]);
    expect(schema.healthRules.find((r) => r.key === "reachability")?.direction).toBe("boolean");
  });
});
