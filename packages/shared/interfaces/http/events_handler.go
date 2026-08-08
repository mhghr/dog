package api

import (
	"fmt"
	"net/http"
	"time"
)

const sseHeartbeatInterval = 15 * time.Second

// eventStream is the SSE gateway: it fans out live probe results to the
// console without polling.
func (h *Handler) eventStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, r, http.StatusInternalServerError, "streaming_unsupported", "Streaming is not supported", nil)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	_, _ = fmt.Fprint(w, "retry: 5000\n\n")
	flusher.Flush()

	subscription, cancel := h.deps.Bus.Subscribe(64)
	defer cancel()

	heartbeatTicker := time.NewTicker(sseHeartbeatInterval)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return

		case <-heartbeatTicker.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()

		case event, open := <-subscription:
			if !open {
				return
			}

			if _, err := fmt.Fprintf(
				w,
				"event: %s\nid: %s\ndata: %s\n\n",
				event.Name, event.ID, event.Data,
			); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
