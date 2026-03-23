package observability

import (
	"log/slog"
	"os"
)

func NewLogger(appEnv string) *slog.Logger {
	level := slog.LevelInfo
	if appEnv == "development" {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}
