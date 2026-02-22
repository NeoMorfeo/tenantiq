package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/neomorfeo/tenantiq/internal/app"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

// TenantResponse is the API representation of a tenant.
type TenantResponse struct {
	ID        string `json:"id" doc:"Unique identifier"`
	Name      string `json:"name" doc:"Display name"`
	Slug      string `json:"slug" doc:"URL-friendly identifier"`
	Status    string `json:"status" doc:"Lifecycle state"`
	Plan      string `json:"plan" doc:"Subscription plan"`
	CreatedAt string `json:"created_at" doc:"Creation timestamp (ISO 8601)"`
	UpdatedAt string `json:"updated_at" doc:"Last update timestamp (ISO 8601)"`
}

func toTenantResponse(t domain.Tenant) TenantResponse {
	return TenantResponse{
		ID:        t.ID,
		Name:      t.Name,
		Slug:      t.Slug,
		Status:    string(t.Status),
		Plan:      t.Plan,
		CreatedAt: t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: t.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// --- Create Tenant ---

type CreateTenantInput struct {
	Body struct {
		Name string `json:"name" minLength:"1" maxLength:"255" doc:"Display name"`
		Slug string `json:"slug" minLength:"1" maxLength:"100" pattern:"^[a-z0-9]+(?:-[a-z0-9]+)*$" doc:"URL-friendly identifier (lowercase, hyphens)"`
		Plan string `json:"plan,omitempty" default:"free" doc:"Subscription plan"`
	}
}

type CreateTenantOutput struct {
	Body TenantResponse
}

// --- Get Tenant ---

type GetTenantInput struct {
	ID string `path:"id" doc:"Tenant ID"`
}

type GetTenantOutput struct {
	Body TenantResponse
}

// --- List Tenants ---

type ListTenantsInput struct {
	Status string `query:"status" required:"false" doc:"Filter by status"`
	Limit  int    `query:"limit" required:"false" default:"50" doc:"Max results"`
	Offset int    `query:"offset" required:"false" default:"0" doc:"Pagination offset"`
}

type ListTenantsOutput struct {
	Body []TenantResponse
}

// --- Transition ---

type TransitionInput struct {
	ID   string `path:"id" doc:"Tenant ID"`
	Body struct {
		Event string `json:"event" doc:"Lifecycle event to trigger" enum:"provision_complete,suspend,reactivate,delete,deletion_complete"`
	}
}

type TransitionOutput struct {
	Body TenantResponse
}

// Register adds all tenant API routes to the Huma API.
func Register(api huma.API, svc *app.TenantService) {
	huma.Register(api, huma.Operation{
		OperationID: "create-tenant",
		Method:      http.MethodPost,
		Path:        "/api/v1/tenants",
		Summary:     "Create a new tenant",
		Tags:        []string{"Tenants"},
	}, func(ctx context.Context, input *CreateTenantInput) (*CreateTenantOutput, error) {
		tenant, err := svc.Create(ctx, input.Body.Name, input.Body.Slug, input.Body.Plan)
		if err != nil {
			return nil, toHumaError(err)
		}
		return &CreateTenantOutput{Body: toTenantResponse(tenant)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-tenant",
		Method:      http.MethodGet,
		Path:        "/api/v1/tenants/{id}",
		Summary:     "Get a tenant by ID",
		Tags:        []string{"Tenants"},
	}, func(ctx context.Context, input *GetTenantInput) (*GetTenantOutput, error) {
		tenant, err := svc.GetByID(ctx, input.ID)
		if err != nil {
			return nil, toHumaError(err)
		}
		return &GetTenantOutput{Body: toTenantResponse(tenant)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-tenants",
		Method:      http.MethodGet,
		Path:        "/api/v1/tenants",
		Summary:     "List tenants",
		Tags:        []string{"Tenants"},
	}, func(ctx context.Context, input *ListTenantsInput) (*ListTenantsOutput, error) {
		filter := domain.ListFilter{
			Limit:  input.Limit,
			Offset: input.Offset,
		}
		if input.Status != "" {
			s := domain.Status(input.Status)
			filter.Status = &s
		}

		tenants, err := svc.List(ctx, filter)
		if err != nil {
			return nil, toHumaError(err)
		}

		resp := make([]TenantResponse, len(tenants))
		for i, t := range tenants {
			resp[i] = toTenantResponse(t)
		}
		return &ListTenantsOutput{Body: resp}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "transition-tenant",
		Method:      http.MethodPost,
		Path:        "/api/v1/tenants/{id}/events",
		Summary:     "Trigger a lifecycle event",
		Tags:        []string{"Tenants"},
	}, func(ctx context.Context, input *TransitionInput) (*TransitionOutput, error) {
		tenant, err := svc.Transition(ctx, input.ID, domain.Event(input.Body.Event))
		if err != nil {
			return nil, toHumaError(err)
		}
		return &TransitionOutput{Body: toTenantResponse(tenant)}, nil
	})
}

// toHumaError translates domain errors to Huma HTTP errors.
func toHumaError(err error) error {
	if errors.Is(err, domain.ErrTenantNotFound) {
		return huma.Error404NotFound("tenant not found")
	}

	var slugErr *domain.SlugConflictError
	if errors.As(err, &slugErr) {
		return huma.Error409Conflict(slugErr.Error())
	}

	var trErr *domain.TransitionError
	if errors.As(err, &trErr) {
		return huma.Error422UnprocessableEntity(trErr.Error())
	}

	return huma.Error500InternalServerError("internal server error")
}
