package logger

import (
	"log/slog"
	"os"
)

func NewLogger(env string) *slog.Logger {
	var logger *slog.Logger

	switch env {
	case "prod":
		logger = slog.New(
			slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}))
	case "dev":
		logger = slog.New(
			slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))
	}

	slog.SetDefault(logger)

	return logger
}
