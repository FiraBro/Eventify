package observability

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
)

type requestIDContextKey struct{}

// ContextWithRequestID associates a validated request ID with a context.
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// RequestIDFromContext returns the request ID attached by the HTTP middleware.
func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

// NewJSONLogger creates a JSON logger with a validated level. An empty level defaults to info.
func NewJSONLogger(level string, output io.Writer) (*slog.Logger, error) {
	var logLevel slog.Level
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "", "info":
		logLevel = slog.LevelInfo
	case "debug":
		logLevel = slog.LevelDebug
	case "warn", "warning":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		return nil, errors.New("LOG_LEVEL must be one of debug, info, warn, or error")
	}
	if output == nil {
		output = io.Discard
	}
	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{Level: logLevel})
	return slog.New(requestContextHandler{next: handler}), nil
}

// requestContextHandler automatically attaches request correlation to all slog calls
// that use a request context, without requiring each call site to add the field manually.
type requestContextHandler struct{ next slog.Handler }

func (h requestContextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h requestContextHandler) Handle(ctx context.Context, record slog.Record) error {
	if requestID := RequestIDFromContext(ctx); requestID != "" {
		record.AddAttrs(slog.String("request_id", requestID))
	}
	return h.next.Handle(ctx, record)
}

func (h requestContextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return requestContextHandler{next: h.next.WithAttrs(attrs)}
}

func (h requestContextHandler) WithGroup(name string) slog.Handler {
	return requestContextHandler{next: h.next.WithGroup(name)}
}
