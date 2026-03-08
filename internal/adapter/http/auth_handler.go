package http

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/neomorfeo/tenantiq/internal/app"
	"github.com/neomorfeo/tenantiq/internal/domain"
)

// --- Login ---

type LoginInput struct {
	Body struct {
		Email    string `json:"email" format:"email" doc:"User email"`
		Password string `json:"password" minLength:"1" doc:"User password"`
	}
}

type LoginOutput struct {
	Body app.TokenPair
}

// --- Refresh ---

type RefreshInput struct {
	Body struct {
		RefreshToken string `json:"refresh_token,omitempty" doc:"Refresh token (optional when using cookies)"`
	}
}

type RefreshOutput struct {
	Body app.TokenPair
}

// --- Me ---

type MeOutput struct {
	Body struct {
		UserID string      `json:"user_id" doc:"Authenticated user ID"`
		Email  string      `json:"email" doc:"Authenticated user email"`
		Role   domain.Role `json:"role" doc:"Authenticated user role"`
	}
}

// RegisterAuth adds authentication routes to the Huma API.
func RegisterAuth(api huma.API, authSvc *app.AuthService, tokens domain.TokenService, cookieCfg CookieConfig) {
	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/login",
		Summary:     "Authenticate with email and password",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *LoginInput) (*LoginOutput, error) {
		pair, err := authSvc.Login(ctx, input.Body.Email, input.Body.Password)
		if err != nil {
			if errors.Is(err, domain.ErrInvalidCredentials) {
				slog.WarnContext(ctx, "login failed: invalid credentials", "email", input.Body.Email)
				return nil, huma.Error401Unauthorized("invalid email or password")
			}
			slog.ErrorContext(ctx, "login failed: internal error", "email", input.Body.Email, "error", err)
			return nil, huma.Error500InternalServerError("internal server error")
		}

		// Set HttpOnly cookies for SPA auth.
		if w := responseWriter(ctx); w != nil {
			setAuthCookies(w, pair.AccessToken, pair.RefreshToken, cookieCfg)
		}

		slog.InfoContext(ctx, "login successful", "email", input.Body.Email)
		return &LoginOutput{Body: pair}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "refresh-token",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/refresh",
		Summary:     "Refresh an access token",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *RefreshInput) (*RefreshOutput, error) {
		refreshToken := input.Body.RefreshToken

		// Fall back to cookie if no token in body (SPA flow).
		if refreshToken == "" {
			refreshToken = cookieValue(ctx, refreshCookieName)
		}
		if refreshToken == "" {
			return nil, huma.Error401Unauthorized("missing refresh token")
		}

		pair, err := authSvc.Refresh(ctx, refreshToken)
		if err != nil {
			if errors.Is(err, domain.ErrUnauthorized) {
				return nil, huma.Error401Unauthorized("invalid or expired refresh token")
			}
			return nil, huma.Error500InternalServerError("internal server error")
		}

		if w := responseWriter(ctx); w != nil {
			setAuthCookies(w, pair.AccessToken, pair.RefreshToken, cookieCfg)
		}

		return &RefreshOutput{Body: pair}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "logout",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/logout",
		Summary:     "Clear authentication cookies",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		if w := responseWriter(ctx); w != nil {
			clearAuthCookies(w, cookieCfg)
		}
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "me",
		Method:      http.MethodGet,
		Path:        "/api/v1/auth/me",
		Summary:     "Get the current authenticated user",
		Tags:        []string{"Auth"},
		Security:    BearerSecurity,
	}, func(ctx context.Context, _ *struct{}) (*MeOutput, error) {
		claims, ok := ClaimsFromContext(ctx)
		if !ok {
			return nil, huma.Error401Unauthorized("not authenticated")
		}
		out := &MeOutput{}
		out.Body.UserID = claims.UserID
		out.Body.Email = claims.Email
		out.Body.Role = claims.Role
		return out, nil
	})
}
