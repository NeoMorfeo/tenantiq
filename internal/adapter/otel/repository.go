package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/neomorfeo/tenantiq/internal/domain"
)

const tracerName = "github.com/neomorfeo/tenantiq/internal/adapter/otel"

// TracingRepository wraps a domain.TenantRepository with OpenTelemetry tracing.
// Each method creates a span with semantic attributes and records errors.
type TracingRepository struct {
	next   domain.TenantRepository
	tracer trace.Tracer
}

// Compile-time check: TracingRepository implements domain.TenantRepository.
var _ domain.TenantRepository = (*TracingRepository)(nil)

// NewTracingRepository creates a tracing decorator around the given repository.
func NewTracingRepository(next domain.TenantRepository) *TracingRepository {
	return &TracingRepository{
		next:   next,
		tracer: otel.Tracer(tracerName),
	}
}

func (r *TracingRepository) Create(ctx context.Context, tenant domain.Tenant) error {
	ctx, span := r.tracer.Start(ctx, "TenantRepository.Create",
		trace.WithAttributes(
			attribute.String("tenant.id", tenant.ID),
			attribute.String("tenant.slug", tenant.Slug),
		),
	)
	defer span.End()

	err := r.next.Create(ctx, tenant)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (r *TracingRepository) GetByID(ctx context.Context, id string) (domain.Tenant, error) {
	ctx, span := r.tracer.Start(ctx, "TenantRepository.GetByID",
		trace.WithAttributes(attribute.String("tenant.id", id)),
	)
	defer span.End()

	tenant, err := r.next.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return tenant, err
}

func (r *TracingRepository) GetBySlug(ctx context.Context, slug string) (domain.Tenant, error) {
	ctx, span := r.tracer.Start(ctx, "TenantRepository.GetBySlug",
		trace.WithAttributes(attribute.String("tenant.slug", slug)),
	)
	defer span.End()

	tenant, err := r.next.GetBySlug(ctx, slug)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return tenant, err
}

func (r *TracingRepository) List(ctx context.Context, filter domain.ListFilter) ([]domain.Tenant, error) {
	ctx, span := r.tracer.Start(ctx, "TenantRepository.List",
		trace.WithAttributes(
			attribute.Int("filter.limit", filter.Limit),
			attribute.Int("filter.offset", filter.Offset),
		),
	)
	defer span.End()

	if filter.Status != nil {
		span.SetAttributes(attribute.String("filter.status", string(*filter.Status)))
	}

	tenants, err := r.next.List(ctx, filter)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetAttributes(attribute.Int("result.count", len(tenants)))
	}
	return tenants, err
}

func (r *TracingRepository) Update(ctx context.Context, tenant domain.Tenant) error {
	ctx, span := r.tracer.Start(ctx, "TenantRepository.Update",
		trace.WithAttributes(
			attribute.String("tenant.id", tenant.ID),
			attribute.String("tenant.status", string(tenant.Status)),
		),
	)
	defer span.End()

	err := r.next.Update(ctx, tenant)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}
