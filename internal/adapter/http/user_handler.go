package http

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/neomorfeo/tenantiq/internal/app"
	"github.com/neomorfeo/tenantiq/internal/domain"
	"github.com/neomorfeo/tenantiq/internal/i18n"
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
			return nil, toUserError(ctx, err)
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
		l := i18n.Localizer(ctx)
		users, err := userSvc.ListUsers(ctx)
		if err != nil {
			return nil, huma.Error500InternalServerError(i18n.T(l, "error.internal", nil))
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
			return nil, toUserError(ctx, err)
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
			return nil, toUserError(ctx, err)
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
			return nil, toUserError(ctx, err)
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
		l := i18n.Localizer(ctx)
		// Users can only change their own password.
		claims, ok := ClaimsFromContext(ctx)
		if !ok {
			return nil, huma.Error401Unauthorized(i18n.T(l, "error.unauthorized", nil))
		}
		if claims.UserID != input.ID {
			return nil, huma.Error403Forbidden(i18n.T(l, "error.forbidden_password", nil))
		}

		err := userSvc.UpdatePassword(ctx, input.ID, input.Body.CurrentPassword, input.Body.NewPassword)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidCredentials) {
				return nil, huma.Error401Unauthorized(i18n.T(l, "error.password_incorrect", nil))
			}
			return nil, toUserError(ctx, err)
		}
		return nil, nil
	})
}

// --- Profile types ---

type ProfileResponse struct {
	ID        string `json:"id" doc:"User ID"`
	Email     string `json:"email" doc:"Email address"`
	Name      string `json:"name" doc:"Display name"`
	Role      string `json:"role" doc:"User role"`
	Theme     string `json:"theme" doc:"UI theme (light, dark, system)"`
	Language  string `json:"language" doc:"UI language (en, es)"`
	CreatedAt string `json:"created_at" doc:"Creation timestamp (ISO 8601)"`
	UpdatedAt string `json:"updated_at" doc:"Last update timestamp (ISO 8601)"`
}

type ProfileOutput struct {
	Body ProfileResponse
}

type UpdateProfileInput struct {
	Body struct {
		Name  *string `json:"name,omitempty" minLength:"1" maxLength:"255" doc:"Display name"`
		Email *string `json:"email,omitempty" format:"email" doc:"Email address"`
	}
}

type UpdateProfileOutput struct {
	Body ProfileResponse
}

type UpdatePreferencesInput struct {
	Body struct {
		Theme    *string `json:"theme,omitempty" enum:"light,dark,system" doc:"UI theme"`
		Language *string `json:"language,omitempty" enum:"en,es" doc:"UI language"`
	}
}

type UpdatePreferencesOutput struct {
	Body struct {
		Theme    string `json:"theme" doc:"UI theme"`
		Language string `json:"language" doc:"UI language"`
	}
}

func toProfileResponse(u domain.User) ProfileResponse {
	return ProfileResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      string(u.Role),
		Theme:     string(u.Theme),
		Language:  string(u.Language),
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: u.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// RegisterProfile adds self-service profile and preference endpoints.
func RegisterProfile(api huma.API, userSvc *app.UserService) {
	huma.Register(api, huma.Operation{
		OperationID: "get-profile",
		Method:      http.MethodGet,
		Path:        "/api/v1/users/me",
		Summary:     "Get own profile",
		Tags:        []string{"Profile"},
		Security:    BearerSecurity,
	}, func(ctx context.Context, _ *struct{}) (*ProfileOutput, error) {
		l := i18n.Localizer(ctx)
		claims, ok := ClaimsFromContext(ctx)
		if !ok {
			return nil, huma.Error401Unauthorized(i18n.T(l, "error.not_authenticated", nil))
		}
		user, err := userSvc.GetUser(ctx, claims.UserID)
		if err != nil {
			return nil, toUserError(ctx, err)
		}
		return &ProfileOutput{Body: toProfileResponse(user)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-profile",
		Method:      http.MethodPatch,
		Path:        "/api/v1/users/me",
		Summary:     "Update own profile",
		Tags:        []string{"Profile"},
		Security:    BearerSecurity,
	}, func(ctx context.Context, input *UpdateProfileInput) (*UpdateProfileOutput, error) {
		l := i18n.Localizer(ctx)
		claims, ok := ClaimsFromContext(ctx)
		if !ok {
			return nil, huma.Error401Unauthorized(i18n.T(l, "error.not_authenticated", nil))
		}
		user, err := userSvc.UpdateProfile(ctx, claims.UserID, input.Body.Name, input.Body.Email)
		if err != nil {
			return nil, toUserError(ctx, err)
		}
		return &UpdateProfileOutput{Body: toProfileResponse(user)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "update-preferences",
		Method:      http.MethodPatch,
		Path:        "/api/v1/users/me/preferences",
		Summary:     "Update own preferences",
		Tags:        []string{"Profile"},
		Security:    BearerSecurity,
	}, func(ctx context.Context, input *UpdatePreferencesInput) (*UpdatePreferencesOutput, error) {
		l := i18n.Localizer(ctx)
		claims, ok := ClaimsFromContext(ctx)
		if !ok {
			return nil, huma.Error401Unauthorized(i18n.T(l, "error.not_authenticated", nil))
		}

		var theme *domain.Theme
		if input.Body.Theme != nil {
			t := domain.Theme(*input.Body.Theme)
			theme = &t
		}
		var language *domain.Language
		if input.Body.Language != nil {
			lang := domain.Language(*input.Body.Language)
			language = &lang
		}

		user, err := userSvc.UpdatePreferences(ctx, claims.UserID, theme, language)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidPreference) {
				return nil, huma.Error422UnprocessableEntity(i18n.T(l, "validation.failed", nil))
			}
			return nil, toUserError(ctx, err)
		}

		out := &UpdatePreferencesOutput{}
		out.Body.Theme = string(user.Theme)
		out.Body.Language = string(user.Language)
		return out, nil
	})
}

// toUserError translates domain errors to Huma HTTP errors.
func toUserError(ctx context.Context, err error) error {
	l := i18n.Localizer(ctx)

	if errors.Is(err, domain.ErrUserNotFound) {
		return huma.Error404NotFound(i18n.T(l, "error.user_not_found", nil))
	}
	if errors.Is(err, domain.ErrSelfDelete) {
		return huma.Error409Conflict(i18n.T(l, "error.self_delete", nil))
	}

	var emailErr *domain.EmailConflictError
	if errors.As(err, &emailErr) {
		return huma.Error409Conflict(i18n.T(l, "error.email_conflict", map[string]string{"Email": emailErr.Email}))
	}

	return huma.Error500InternalServerError(i18n.T(l, "error.internal", nil))
}
