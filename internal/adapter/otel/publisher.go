package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/neomorfeo/tenantiq/internal/domain"
)

// TracingPublisher wraps a domain.EventPublisher with OpenTelemetry tracing.
type TracingPublisher struct {
	next   domain.EventPublisher
	tracer trace.Tracer
}

// Compile-time check: TracingPublisher implements domain.EventPublisher.
var _ domain.EventPublisher = (*TracingPublisher)(nil)

// NewTracingPublisher creates a tracing decorator around the given publisher.
func NewTracingPublisher(next domain.EventPublisher) *TracingPublisher {
	return &TracingPublisher{
		next:   next,
		tracer: otel.Tracer(tracerName),
	}
}

func (p *TracingPublisher) Publish(ctx context.Context, event domain.Event, tenant domain.Tenant) error {
	ctx, span := p.tracer.Start(ctx, "EventPublisher.Publish",
		trace.WithAttributes(
			attribute.String("event.type", string(event)),
			attribute.String("tenant.id", tenant.ID),
			attribute.String("tenant.slug", tenant.Slug),
		),
	)
	defer span.End()

	err := p.next.Publish(ctx, event, tenant)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}
