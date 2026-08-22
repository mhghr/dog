import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";

import { resourcesApi } from "@/entities/resource/api/resource.api";
import {
  useResource,
  useResourceMonitorMetrics,
  useResourceMonitors,
  useResourceMonitorResultAt,
  useResources,
  useResourceTypes,
} from "./use-resource";

vi.mock("@/entities/resource/api/resource.api", () => ({
  resourcesApi: {
    listTypes: vi.fn(),
    listMonitorTypes: vi.fn(),
    list: vi.fn(),
    getById: vi.fn(),
    listMonitors: vi.fn(),
    getMonitorMetrics: vi.fn(),
  },
}));

function wrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}

const mockedApi = vi.mocked(resourcesApi);

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("useResourceTypes", () => {
  it("fetches the resource types", async () => {
    mockedApi.listTypes.mockResolvedValue({ items: [] } as never);
    const { result } = renderHook(() => useResourceTypes(), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockedApi.listTypes).toHaveBeenCalledTimes(1);
  });
});

describe("useResources", () => {
  it("fetches with the serialized query string", async () => {
    mockedApi.list.mockResolvedValue({ items: [] } as never);
    const { result } = renderHook(() => useResources({ page: 2, search: "web" }), {
      wrapper: wrapper(),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockedApi.list).toHaveBeenCalledWith(
      expect.stringContaining("page=2"),
    );
    expect(mockedApi.list).toHaveBeenCalledWith(
      expect.stringContaining("search=web"),
    );
  });
});

describe("useResource", () => {
  it("stays disabled without an id", async () => {
    const { result } = renderHook(() => useResource(undefined), { wrapper: wrapper() });
    expect(result.current.isFetching).toBe(false);
    expect(mockedApi.getById).not.toHaveBeenCalled();
  });

  it("fetches the resource when an id is present", async () => {
    mockedApi.getById.mockResolvedValue({ id: "r1" } as never);
    const { result } = renderHook(() => useResource("r1"), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockedApi.getById).toHaveBeenCalledWith("r1");
  });
});

describe("useResourceMonitors", () => {
  it("fetches monitors for the resource", async () => {
    mockedApi.listMonitors.mockResolvedValue({ items: [] } as never);
    const { result } = renderHook(() => useResourceMonitors("r1"), { wrapper: wrapper() });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockedApi.listMonitors).toHaveBeenCalledWith("r1");
  });
});

describe("useResourceMonitorMetrics", () => {
  it("fetches metrics with the range", async () => {
    mockedApi.getMonitorMetrics.mockResolvedValue({ series: [], latest: [] } as never);
    const { result } = renderHook(() => useResourceMonitorMetrics("r1", "m1", "1h"), {
      wrapper: wrapper(),
    });
    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockedApi.getMonitorMetrics).toHaveBeenCalledWith(
      "r1",
      "m1",
      expect.any(String),
    );
  });

  it("stays disabled without monitorId", async () => {
    const { result } = renderHook(() => useResourceMonitorMetrics("r1", undefined, "1h"), {
      wrapper: wrapper(),
    });
    expect(result.current.isFetching).toBe(false);
  });
});

describe("useResourceMonitorResultAt", () => {
  it("selects the drill-down result", async () => {
    const selected = { id: "p1", monitor_id: "m1" };
    mockedApi.getMonitorMetrics.mockResolvedValue({ selected } as never);
    const { result } = renderHook(
      () => useResourceMonitorResultAt("r1", "m1", "1h", "2026-01-01T00:00:00Z"),
      { wrapper: wrapper() },
    );
    await waitFor(() => expect(result.current.data).toEqual(selected));
  });
});
