package bielog

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type LoggerCtxKey struct{}

func NewLogger(loggerType string, logLevel string, handlerOpts *slog.HandlerOptions) *slog.Logger {
	var handler slog.Handler

	level := parseLogLevel(logLevel)

	var resultingOpts slog.HandlerOptions
	if handlerOpts != nil {
		resultingOpts = *handlerOpts
	} else {
		resultingOpts = slog.HandlerOptions{}
	}
	resultingOpts.Level = level

	switch loggerType {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, &resultingOpts)
	case "text":
		handler = slog.NewTextHandler(os.Stdout, &resultingOpts)
	default:
		handler = slog.NewTextHandler(os.Stdout, &resultingOpts)
	}

	return slog.New(handler)
}

func FromCtx(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(LoggerCtxKey{}).(*slog.Logger); ok {
		return logger
	}
	// Fallback to a default logger
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func CtxWithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, LoggerCtxKey{}, logger)
}

// parseLogLevel converts a string to slog.Level
func parseLogLevel(levelStr string) slog.Level {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo // Default level
	}
}
