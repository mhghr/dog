import { QueryClient } from "@tanstack/react-query";

// Cache policy knobs (seconds). Static reference data is cached for a long
// time; normal operational data for tens of seconds; realtime data streams
// through SSE/WebSocket instead of polling.
export const cachePolicy = {
  static: {
    staleTime: 60 * 60 * 1000, // 1h — resource types, plugin schemas
    gcTime: 24 * 60 * 60 * 1000, // 24h
  },
  normal: {
    staleTime: 30_000, // 30s — resources, monitors
    gcTime: 5 * 60 * 1000,
  },
  live: {
    staleTime: 10_000, // 10s — dashboards, status feeds
    gcTime: 60_000,
  },
  realtime: {
    staleTime: 0, // streamed via SSE/WebSocket, no polling
    gcTime: 30_000,
  },
} as const;

export function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: cachePolicy.normal.staleTime,
        refetchOnWindowFocus: true,
        retry: 1,
      },
    },
  });
}
