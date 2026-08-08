package logging

import (
	"log/slog"
	"os"
	"strings"
)

func New(level, format, service string) *slog.Logger {
	options := &slog.HandlerOptions{
		Level: parseLevel(level),
	}

	var handler slog.Handler
	if strings.EqualFold(format, "text") {
		handler = slog.NewTextHandler(os.Stdout, options)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, options)
	}

	return slog.New(handler).With("service", service)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
