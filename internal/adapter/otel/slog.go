package otel

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel/trace"
)

// NewTracedHandler wraps a slog.Handler to inject trace_id and span_id
// from the OTel span in context into every log record.
func NewTracedHandler(inner slog.Handler) slog.Handler {
	return &tracedHandler{inner: inner}
}

type tracedHandler struct {
	inner slog.Handler
}

func (h *tracedHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *tracedHandler) Handle(ctx context.Context, r slog.Record) error {
	sc := trace.SpanFromContext(ctx).SpanContext()
	if sc.IsValid() {
		r.AddAttrs(
			slog.String("trace_id", sc.TraceID().String()),
			slog.String("span_id", sc.SpanID().String()),
		)
	}
	return h.inner.Handle(ctx, r)
}

func (h *tracedHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &tracedHandler{inner: h.inner.WithAttrs(attrs)}
}

func (h *tracedHandler) WithGroup(name string) slog.Handler {
	return &tracedHandler{inner: h.inner.WithGroup(name)}
}

// NewMultiHandler fans out log records to multiple handlers.
func NewMultiHandler(handlers ...slog.Handler) slog.Handler {
	return &multiHandler{handlers: handlers}
}

type multiHandler struct {
	handlers []slog.Handler
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			_ = handler.Handle(ctx, r)
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}
