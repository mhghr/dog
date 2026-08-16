"use client";

import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { SseClient } from "@/platform/realtime/sse-client";
import { REAL_TIME_EVENTS } from "@/platform/realtime/events";
import { invalidateActive, throttleInvalidate } from "@/shared/data/realtime";
import type { LiveProbeEvent } from "@/entities/monitor/model/result";

// useLiveResults subscribes to the SSE gateway and incrementally refreshes
// active queries without full page reloads. Invalidation is throttled so a
// busy stream cannot stampede the API. The stream is authenticated via the
// HttpOnly session cookie.
export function useLiveResults(enabled = true) {
  const queryClient = useQueryClient();
  const lastInvalidatedAt = useRef(0);

  useEffect(() => {
    if (!enabled) {
      return;
    }

    const client = new SseClient("/events/stream");
    client.connect();

    const off = client.on(REAL_TIME_EVENTS.probeResult, (event) => {
      let payload: LiveProbeEvent;
      try {
        payload = JSON.parse(event.data) as LiveProbeEvent;
      } catch {
        return;
      }

      if (
        !throttleInvalidate(queryClient, {
          lastInvalidatedAt: lastInvalidatedAt.current,
        })
      ) {
        return;
      }
      lastInvalidatedAt.current = Date.now();

      invalidateActive(queryClient, ["monitors", "list"]);
      invalidateActive(queryClient, ["dashboard"]);
      invalidateActive(queryClient, ["monitors", payload.monitor_id]);

      // Resource-scoped monitor queries (["resources", <id>, "monitors", ...])
      // share the legacy monitor id; invalidate the active per-monitor metrics/
      // results queries so live probe results flow into the monitoring view
      // without a full page reload.
      void queryClient.invalidateQueries({
        predicate: (query) => {
          const key = query.queryKey;
          if (!Array.isArray(key) || key.length < 3) return false;
          if (key[0] !== "resources" || key[2] !== "monitors") return false;
          return key.length === 3 || key.includes(payload.monitor_id);
        },
        refetchType: "active",
      });
    });

    return () => {
      off();
      client.close();
    };
  }, [queryClient, enabled]);
}
