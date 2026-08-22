import { describe, expect, it } from "vitest";

import { consoleBasePath } from "./console-base";

describe("consoleBasePath", () => {
  it("uses the /app prefix for classic console routes", () => {
    expect(consoleBasePath("/app/dashboard")).toBe("/app");
    expect(consoleBasePath("/app/resources")).toBe("/app");
    expect(consoleBasePath("/")).toBe("/app");
    expect(consoleBasePath("")).toBe("/app");
  });

  it("uses the /app prefix for non-console paths", () => {
    expect(consoleBasePath("/login")).toBe("/app");
    expect(consoleBasePath("/api/health")).toBe("/app");
  });

  it("extracts the workspace slug prefix", () => {
    expect(consoleBasePath("/console/w/acme/dashboard")).toBe("/console/w/acme");
    expect(consoleBasePath("/console/w/acme/resources/r1")).toBe("/console/w/acme");
  });

  it("handles nested workspace slugs with hyphens", () => {
    expect(consoleBasePath("/console/w/my-workspace/alerts")).toBe("/console/w/my-workspace");
  });
});
