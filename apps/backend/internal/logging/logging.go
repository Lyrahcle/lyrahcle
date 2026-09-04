package logging

import (
	"context"
	"io"
	"log"
	"log/slog"

	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/otel/trace"

	"github.com/mlko01/lyrahcle/apps/backend/internal/auth"
)

func Configure(output io.Writer) *slog.Logger {
	handler := slog.NewJSONHandler(output, nil)
	logger := slog.New(handler)
	slog.SetDefault(logger)
	log.SetFlags(0)
	log.SetOutput(slog.NewLogLogger(handler, slog.LevelInfo).Writer())
	return logger
}

func FromContext(ctx context.Context) *slog.Logger {
	logger := slog.Default()
	if requestID := middleware.GetReqID(ctx); requestID != "" {
		logger = logger.With("request_id", requestID)
	}
	if span := trace.SpanContextFromContext(ctx); span.IsValid() {
		logger = logger.With("trace_id", span.TraceID().String())
	}
	if claims, ok := auth.FromContext(ctx); ok {
		logger = logger.With("tenant_id", claims.OrganizationID.String(), "user_id", claims.UserID.String())
	}
	if agentID, ok := auth.AgentIDFromContext(ctx); ok {
		logger = logger.With("agent_id", agentID)
	}
	return logger
}
