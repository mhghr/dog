import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { resourcesApi } from "./resource.api";

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

const fetchMock = vi.fn();

const lastCall = () => {
  const [input, init] = fetchMock.mock.calls[fetchMock.mock.calls.length - 1] as [
    RequestInfo | URL,
    RequestInit | undefined,
  ];
  return { url: String(input), method: (init?.method ?? "GET") as string };
};

describe("resourcesApi", () => {
  beforeEach(() => {
    fetchMock.mockReset();
    vi.stubGlobal("fetch", fetchMock);
    fetchMock.mockResolvedValue(jsonResponse(200, {}));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("listTypes GETs the resource-types endpoint", async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { items: [] }));
    await resourcesApi.listTypes();
    expect(lastCall().url).toBe("/api/resource-types");
    expect(lastCall().method).toBe("GET");
  });

  it("listMonitorTypes GETs the monitor-types endpoint", async () => {
    await resourcesApi.listMonitorTypes();
    expect(lastCall().url).toBe("/api/monitor-types");
  });

  it("overview GETs the resources overview endpoint", async () => {
    await resourcesApi.overview();
    expect(lastCall().url).toBe("/api/resources/overview");
  });

  it("list appends the query string", async () => {
    await resourcesApi.list("page=1&page_size=20");
    expect(lastCall().url).toBe("/api/resources?page=1&page_size=20");
  });

  it("getById hits the resource detail endpoint", async () => {
    await resourcesApi.getById("r1");
    expect(lastCall().url).toBe("/api/resources/r1");
  });

  it("create POSTs the input body", async () => {
    const input = { name: "api", resource_type_id: "rt1" };
    await resourcesApi.create(input);
    expect(lastCall().url).toBe("/api/resources");
    expect(lastCall().method).toBe("POST");
    const body = JSON.parse(fetchMock.mock.calls[0][1].body as string);
    expect(body).toEqual(input);
  });

  it("update PUTs to the resource endpoint", async () => {
    await resourcesApi.update("r1", { name: "renamed" });
    expect(lastCall().url).toBe("/api/resources/r1");
    expect(lastCall().method).toBe("PUT");
  });

  it("delete DELETEs the resource endpoint", async () => {
    fetchMock.mockResolvedValue(new Response(null, { status: 204 }));
    await resourcesApi.delete("r1");
    expect(lastCall().url).toBe("/api/resources/r1");
    expect(lastCall().method).toBe("DELETE");
  });

  it("listMonitors hits the resource monitors endpoint", async () => {
    await resourcesApi.listMonitors("r1");
    expect(lastCall().url).toBe("/api/resources/r1/monitors");
  });

  it("createMonitor POSTs to the resource monitors endpoint", async () => {
    const input = { name: "m", monitor_type_id: "mt1", interval_seconds: 30, timeout_millis: 5000, retries: 2 };
    await resourcesApi.createMonitor("r1", input);
    expect(lastCall().url).toBe("/api/resources/r1/monitors");
    expect(lastCall().method).toBe("POST");
  });

  it("updateMonitor PUTs to the monitor endpoint", async () => {
    const input = { name: "m", monitor_type_id: "mt1", interval_seconds: 30, timeout_millis: 5000, retries: 2 };
    await resourcesApi.updateMonitor("r1", "m1", input);
    expect(lastCall().url).toBe("/api/resources/r1/monitors/m1");
    expect(lastCall().method).toBe("PUT");
  });

  it("getMonitorMetrics appends the query string", async () => {
    await resourcesApi.getMonitorMetrics("r1", "m1", "step=auto");
    expect(lastCall().url).toBe("/api/resources/r1/monitors/m1/metrics?step=auto");
  });

  it("snmpDiscover POSTs to the discover endpoint", async () => {
    await resourcesApi.snmpDiscover("r1", "m1");
    expect(lastCall().url).toBe("/api/resources/r1/monitors/m1/snmp/discover");
    expect(lastCall().method).toBe("POST");
  });

  it("snmpGetTask GETs the task endpoint", async () => {
    await resourcesApi.snmpGetTask("task-1");
    expect(lastCall().url).toBe("/api/snmp/tasks/task-1");
  });

  it("snmpListEvents appends the limit param", async () => {
    await resourcesApi.snmpListEvents("r1", "m1", 10);
    expect(lastCall().url).toBe("/api/resources/r1/monitors/m1/snmp/events?limit=10");
  });

  it("snmpUpdateInterface PUTs to the per-interface endpoint", async () => {
    await resourcesApi.snmpUpdateInterface("r1", "m1", 3, { display_name: "uplink" });
    expect(lastCall().url).toBe("/api/resources/r1/monitors/m1/snmp/interfaces/3");
    expect(lastCall().method).toBe("PUT");
  });

  it("returns the decoded JSON payload", async () => {
    fetchMock.mockResolvedValue(jsonResponse(200, { items: [{ id: "r1" }] }));
    const result = await resourcesApi.list("page=1");
    expect(result.items).toHaveLength(1);
  });

  it("rejects with an error on failure responses", async () => {
    fetchMock.mockResolvedValue(
      jsonResponse(500, { error: { code: "internal", message: "boom" } }),
    );
    await expect(resourcesApi.list("page=1")).rejects.toMatchObject({ status: 500 });
  });
});
