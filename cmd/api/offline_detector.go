package main

import (
	"context"
	"log/slog"
	"time"

	"monitoring-platform/internal/agents"
)

func watchOfflineAgents(ctx context.Context, agentRepo *agents.Repository, logger *slog.Logger) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			markCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			_, err := agentRepo.MarkOfflineAgents(markCtx, 30*time.Second)
			cancel()
			if err != nil {
				logger.Warn("mark offline agents failed", "error", err)
			}
		}
	}
}
