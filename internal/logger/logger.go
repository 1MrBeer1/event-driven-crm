package logger

import (
	"log/slog"
	"os"
	"strings"
)

func New(env, service string) *slog.Logger {
	level := slog.LevelInfo
	if strings.EqualFold(env, "local") || strings.EqualFold(env, "dev") {
		level = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(handler).With("service", service, "env", env)
}
