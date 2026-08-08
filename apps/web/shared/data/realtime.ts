import { useQueryClient } from "@tanstack/react-query";

// Query invalidation helpers for realtime streams. Invalidation is throttled
// so a busy stream cannot stampede the API.
const INVALIDATE_THROTTLE_MS = 3000;

export function throttleInvalidate(
  queryClient: ReturnType<typeof useQueryClient>,
  state: { lastInvalidatedAt: number },
) {
  const now = Date.now();
  if (now - state.lastInvalidatedAt < INVALIDATE_THROTTLE_MS) {
    return false;
  }
  state.lastInvalidatedAt = now;
  return true;
}

export function invalidateActive(
  queryClient: ReturnType<typeof useQueryClient>,
  queryKey: unknown[],
) {
  void queryClient.invalidateQueries({
    queryKey,
    refetchType: "active",
  });
}
