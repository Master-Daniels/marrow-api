package logger

import (
	"log/slog"
	"os"
)

type LoggerService struct {
	level        slog.Level
	isProduction bool
}

func NewLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func NewLoggerFromService(loggerService *LoggerService) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     loggerService.level,
		AddSource: loggerService.isProduction,
	}

	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
