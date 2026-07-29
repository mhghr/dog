"use client";

import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";

import type { LiveProbeEvent } from "@/types/result";

const INVALIDATE_THROTTLE_MS = 3000;

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

    let source: EventSource | null = null;

    try {
      source = new EventSource("/events/v1/stream", {
        withCredentials: true,
      });
    } catch {
      return;
    }

    const onProbeResult = (event: MessageEvent<string>) => {
      let payload: LiveProbeEvent;
      try {
        payload = JSON.parse(event.data) as LiveProbeEvent;
      } catch {
        return;
      }

      const now = Date.now();
      if (now - lastInvalidatedAt.current < INVALIDATE_THROTTLE_MS) {
        return;
      }
      lastInvalidatedAt.current = now;

      void queryClient.invalidateQueries({
        queryKey: ["monitors", "list"],
        refetchType: "active",
      });
      void queryClient.invalidateQueries({
        queryKey: ["dashboard"],
        refetchType: "active",
      });
      void queryClient.invalidateQueries({
        queryKey: ["monitors", payload.monitor_id],
        refetchType: "active",
      });
    };

    source.addEventListener("probe-result", onProbeResult as EventListener);

    return () => {
      source?.removeEventListener("probe-result", onProbeResult as EventListener);
      source?.close();
    };
  }, [queryClient, enabled]);
}
