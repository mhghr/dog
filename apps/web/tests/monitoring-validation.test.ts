import { describe, expect, it } from "vitest";

import {
  hasErrors,
  isValidHostname,
  isValidPort,
  isValidUrl,
  parseStatusCodes,
  validateMonitoringConfig,
} from "@/entities/resource/ui/settings/validation";

function validate(type: string, config: Record<string, unknown>, overrides: Partial<{ target: string; intervalSeconds: number; timeoutMillis: number; retries: number }> = {}) {
  return validateMonitoringConfig({
    type,
    target: overrides.target ?? "api.example.com",
    config,
    intervalSeconds: overrides.intervalSeconds ?? 60,
    timeoutMillis: overrides.timeoutMillis ?? 5000,
    retries: overrides.retries ?? 1,
  });
}

describe("validateMonitoringConfig", () => {
  it("accepts valid configurations for every type", () => {
    const cases: Array<[string, Record<string, unknown>]> = [
      ["ping", { count: 4 }],
      ["http", { method: "GET", expected_status_codes: "200, 204", url: "https://api.example.com" }],
      ["tcp", { port: 443 }],
      ["dns", { record_type: "A", resolver: "" }],
      ["tls", { port: 443, server_name: "" }],
    ];
    for (const [type, config] of cases) {
      expect(hasErrors(validate(type, config)), `${type} should be valid`).toBe(false);
    }
  });

  it("rejects an empty target", () => {
    const errors = validate("ping", { count: 4 }, { target: "" });
    expect(errors.target).toBeDefined();
  });

  it("rejects an invalid TCP port", () => {
    for (const port of [0, 70000, -1]) {
      const errors = validate("tcp", { port });
      expect(errors.port, `port ${port} should fail`).toBeDefined();
    }
  });

  it("rejects an invalid HTTP method and bad status codes", () => {
    const methodErrors = validate("http", { method: "FETCH", expected_status_codes: "200" });
    expect(methodErrors.method).toBeDefined();

    const codeErrors = validate("http", { method: "GET", expected_status_codes: "abc, xyz" });
    expect(codeErrors.expected_status_codes).toBeDefined();
  });

  it("rejects an invalid DNS record type and resolver", () => {
    const typeErrors = validate("dns", { record_type: "SRV" });
    expect(typeErrors.record_type).toBeDefined();

    const resolverErrors = validate("dns", { record_type: "A", resolver: "bad host:53" });
    expect(resolverErrors.resolver).toBeDefined();
  });

  it("rejects an invalid TLS port and SNI hostname", () => {
    const portErrors = validate("tls", { port: 0 });
    expect(portErrors.port).toBeDefined();

    const sniErrors = validate("tls", { port: 443, server_name: "not a hostname!" });
    expect(sniErrors.server_name).toBeDefined();
  });

  it("rejects an invalid ping packet count", () => {
    const errors = validate("ping", { count: 0 });
    expect(errors.count).toBeDefined();
  });

  it("enforces common execution constraints", () => {
    const errors = validate("ping", { count: 4 }, { intervalSeconds: 0, timeoutMillis: 10, retries: 99 });
    expect(errors.interval_seconds).toBeDefined();
    expect(errors.timeout_millis).toBeDefined();
    expect(errors.retries).toBeDefined();
  });
});

describe("helpers", () => {
  it("validates ports", () => {
    expect(isValidPort(443)).toBe(true);
    expect(isValidPort(0)).toBe(false);
    expect(isValidPort(65536)).toBe(false);
    expect(isValidPort("8080")).toBe(true);
  });

  it("validates hostnames and IPs", () => {
    expect(isValidHostname("api.example.com")).toBe(true);
    expect(isValidHostname("localhost")).toBe(true);
    expect(isValidHostname("1.2.3.4")).toBe(true);
    expect(isValidHostname("2001:db8::1")).toBe(true);
    expect(isValidHostname("bad hostname!")).toBe(false);
  });

  it("validates URLs", () => {
    expect(isValidUrl("https://api.example.com")).toBe(true);
    expect(isValidUrl("http://example.com/path")).toBe(true);
    expect(isValidUrl("ftp://example.com")).toBe(false);
    expect(isValidUrl("not a url")).toBe(false);
  });

  it("parses expected status codes", () => {
    expect(parseStatusCodes("200, 204")).toEqual([200, 204]);
    expect(parseStatusCodes("200")).toEqual([200]);
    expect(parseStatusCodes("abc")).toBeNull();
    expect(parseStatusCodes("")).toBeNull();
  });
});
