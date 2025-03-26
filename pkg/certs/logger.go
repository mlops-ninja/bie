package certs

import (
	"context"
	"log/slog"
)

// LoggerAdapter adapts slog.Logger to certs.Logger interface
type LoggerAdapter struct {
	logger *slog.Logger
	ctx    context.Context
}

// NewLoggerAdapter creates a new logger adapter
func NewLoggerAdapter(logger *slog.Logger, ctx context.Context) *LoggerAdapter {
	return &LoggerAdapter{
		logger: logger,
		ctx:    ctx,
	}
}

// Errorf implements certs.Logger interface
func (l *LoggerAdapter) Errorf(format string, args ...interface{}) {
	l.logger.ErrorContext(l.ctx, format, args...)
}

// Infof implements certs.Logger interface
func (l *LoggerAdapter) Infof(format string, args ...interface{}) {
	l.logger.InfoContext(l.ctx, format, args...)
}
