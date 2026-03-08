package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/neomorfeo/tenantiq/internal/app"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

// --- User Response ---

type UserResponse struct {
	ID        string `json:"id" doc:"Unique identifier"`
	Email     string `json:"email" doc:"Email address"`
	Name      string `json:"name" doc:"Display name"`
	Role      string `json:"role" doc:"User role (superadmin, admin, viewer)"`
	CreatedAt string `json:"created_at" doc:"Creation timestamp (ISO 8601)"`
	UpdatedAt string `json:"updated_at" doc:"Last update timestamp (ISO 8601)"`
}

func toUserResponse(u domain.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      string(u.Role),
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// --- Create User ---

type CreateUserInput struct {
	Body struct {
		Email    string `json:"email" format:"email" doc:"Email address"`
		Name     string `json:"name" minLength:"1" maxLength:"255" doc:"Display name"`
		Password string `json:"password" minLength:"8" maxLength:"128" doc:"Password (min 8 chars)"`
		Role     string `json:"role" enum:"superadmin,admin,viewer" doc:"User role"`
	}
}

type CreateUserOutput struct {
	Body UserResponse
}

// --- Get User ---

type GetUserInput struct {
	ID string `path:"id" doc:"User ID"`
}

type GetUserOutput struct {
	Body UserResponse
}

// --- List Users ---

type ListUsersOutput struct {
	Body []UserResponse
}

// --- Update User (PATCH) ---

type UpdateUserInput struct {
	ID   string `path:"id" doc:"User ID"`
	Body struct {
		Name *string `json:"name,omitempty" maxLength:"255" doc:"Display name"`
		Role *string `json:"role,omitempty" enum:"superadmin,admin,viewer" doc:"User role"`
	}
}

type UpdateUserOutput struct {
	Body UserResponse
}

// --- Delete User ---

type DeleteUserInput struct {
	ID string `path:"id" doc:"User ID"`
}

// --- Update Password ---

type UpdatePasswordInput struct {
	ID   string `path:"id" doc:"User ID"`
	Body struct {
		CurrentPassword string `json:"current_password" minLength:"1" doc:"Current password"`
		NewPassword     string `json:"new_password" minLength:"8" maxLength:"128" doc:"New password (min 8 chars)"`
	}
}

// superadminSecurity requires the superadmin role.
var superadminSecurity = RoleSecurity(domain.RoleSuperAdmin)

// RegisterUsers adds user management routes to the Huma API.
// Auth and role enforcement happen via the Huma auth middleware
// using the Security field on each operation.
func RegisterUsers(api huma.API, userSvc *app.UserService) {
	huma.Register(api, huma.Operation{
		OperationID: "create-user",
		Method:      http.MethodPost,
		Path:        "/api/v1/users",
		Summary:     "Create a new user",
		Tags:        []string{"Users"},
		Security:    superadminSecurity,
	}, func(ctx context.Context, input *CreateUserInput) (*CreateUserOutput, error) {
		user, err := userSvc.CreateUser(ctx, input.Body.Email, input.Body.Name, input.Body.Password, domain.Role(input.Body.Role))
		if err != nil {
			return nil, toUserError(err)
		}
		return &CreateUserOutput{Body: toUserResponse(user)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "list-users",
		Method:      http.MethodGet,
		Path:        "/api/v1/users",
		Summary:     "List all users",
		Tags:        []string{"Users"},
		Security:    superadminSecurity,
	}, func(ctx context.Context, _ *struct{}) (*ListUsersOutput, error) {
		users, err := userSvc.ListUsers(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError("internal server error")
		}

		resp := make([]UserResponse, len(users))
		for i, u := range users {
			resp[i] = toUserResponse(u)
		}
		return &ListUsersOutput{Body: resp}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-user",
		Method:      http.MethodGet,
		Path:        "/api/v1/users/{id}",
		Summary:     "Get a user by ID",
		Tags:        []string{"Users"},
		Security:    superadminSecurity,
	}, func(ctx context.Context, input *GetUserInput) (*GetUserOutput, error) {
		user, err := userSvc.GetUser(ctx, input.ID)
		if err != nil {
			return nil, toUserError(err)
		}
		return &GetUserOutput{Body: toUserResponse(user)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-user",
		Method:      http.MethodPatch,
		Path:        "/api/v1/users/{id}",
		Summary:     "Update a user's name or role",
		Tags:        []string{"Users"},
		Security:    superadminSecurity,
	}, func(ctx context.Context, input *UpdateUserInput) (*UpdateUserOutput, error) {
		var role *domain.Role
		if input.Body.Role != nil {
			r := domain.Role(*input.Body.Role)
			role = &r
		}

		user, err := userSvc.UpdateUser(ctx, input.ID, input.Body.Name, role)
		if err != nil {
			return nil, toUserError(err)
		}
		return &UpdateUserOutput{Body: toUserResponse(user)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "delete-user",
		Method:        http.MethodDelete,
		Path:          "/api/v1/users/{id}",
		Summary:       "Delete a user",
		Tags:          []string{"Users"},
		Security:      superadminSecurity,
		DefaultStatus: 204,
	}, func(ctx context.Context, input *DeleteUserInput) (*struct{}, error) {
		claims, _ := ClaimsFromContext(ctx)
		if err := userSvc.DeleteUser(ctx, input.ID, claims.UserID); err != nil {
			return nil, toUserError(err)
		}
		return nil, nil
	})
}

// RegisterPasswordUpdate adds the password update endpoint.
// Requires authentication (any role) — the handler enforces self-only access.
func RegisterPasswordUpdate(api huma.API, userSvc *app.UserService) {
	huma.Register(api, huma.Operation{
		OperationID: "update-password",
		Method:      http.MethodPut,
		Path:        "/api/v1/users/{id}/password",
		Summary:     "Update own password",
		Tags:        []string{"Users"},
		Security:    BearerSecurity,
	}, func(ctx context.Context, input *UpdatePasswordInput) (*struct{}, error) {
		// Users can only change their own password.
		claims, ok := ClaimsFromContext(ctx)
		if !ok {
			return nil, huma.Error401Unauthorized("unauthorized")
		}
		if claims.UserID != input.ID {
			return nil, huma.Error403Forbidden("can only change own password")
		}

		err := userSvc.UpdatePassword(ctx, input.ID, input.Body.CurrentPassword, input.Body.NewPassword)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidCredentials) {
				return nil, huma.Error401Unauthorized("current password is incorrect")
			}
			return nil, toUserError(err)
		}
		return nil, nil
	})
}

// toUserError translates domain errors to Huma HTTP errors.
func toUserError(err error) error {
	if errors.Is(err, domain.ErrUserNotFound) {
		return huma.Error404NotFound("user not found")
	}
	if errors.Is(err, domain.ErrSelfDelete) {
		return huma.Error409Conflict(domain.ErrSelfDelete.Error())
	}

	var emailErr *domain.EmailConflictError
	if errors.As(err, &emailErr) {
		return huma.Error409Conflict(emailErr.Error())
	}

	return huma.Error500InternalServerError("internal server error")
}
