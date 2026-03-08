package otel

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/neomorfeo/tenantiq/internal/domain"
)

// TracingUserRepository wraps a domain.UserRepository with OpenTelemetry tracing.
type TracingUserRepository struct {
	next   domain.UserRepository
	tracer trace.Tracer
}

var _ domain.UserRepository = (*TracingUserRepository)(nil)

// NewTracingUserRepository creates a tracing decorator around the given user repository.
func NewTracingUserRepository(next domain.UserRepository) *TracingUserRepository {
	return &TracingUserRepository{
		next:   next,
		tracer: otel.Tracer(tracerName),
	}
}

func (r *TracingUserRepository) Create(ctx context.Context, user domain.User) error {
	ctx, span := r.tracer.Start(ctx, "UserRepository.Create",
		trace.WithAttributes(
			attribute.String("user.id", user.ID),
			attribute.String("user.email", user.Email),
		),
	)
	defer span.End()

	err := r.next.Create(ctx, user)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (r *TracingUserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	ctx, span := r.tracer.Start(ctx, "UserRepository.GetByID",
		trace.WithAttributes(attribute.String("user.id", id)),
	)
	defer span.End()

	user, err := r.next.GetByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return user, err
}

func (r *TracingUserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	ctx, span := r.tracer.Start(ctx, "UserRepository.GetByEmail",
		trace.WithAttributes(attribute.String("user.email", email)),
	)
	defer span.End()

	user, err := r.next.GetByEmail(ctx, email)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return user, err
}

func (r *TracingUserRepository) List(ctx context.Context) ([]domain.User, error) {
	ctx, span := r.tracer.Start(ctx, "UserRepository.List")
	defer span.End()

	users, err := r.next.List(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetAttributes(attribute.Int("result.count", len(users)))
	}
	return users, err
}

func (r *TracingUserRepository) Update(ctx context.Context, user domain.User) error {
	ctx, span := r.tracer.Start(ctx, "UserRepository.Update",
		trace.WithAttributes(
			attribute.String("user.id", user.ID),
			attribute.String("user.role", string(user.Role)),
		),
	)
	defer span.End()

	err := r.next.Update(ctx, user)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

func (r *TracingUserRepository) Count(ctx context.Context) (int, error) {
	ctx, span := r.tracer.Start(ctx, "UserRepository.Count")
	defer span.End()

	count, err := r.next.Count(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetAttributes(attribute.Int("result.count", count))
	}
	return count, err
}
