package http

import (
	"context"
	"errors"
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
		RefreshToken string `json:"refresh_token" minLength:"1" doc:"Refresh token"`
	}
}

type RefreshOutput struct {
	Body app.TokenPair
}

// RegisterAuth adds authentication routes to the Huma API.
func RegisterAuth(api huma.API, authSvc *app.AuthService) {
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
				return nil, huma.Error401Unauthorized("invalid email or password")
			}
			return nil, huma.Error500InternalServerError("internal server error")
		}
		return &LoginOutput{Body: pair}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "refresh-token",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/refresh",
		Summary:     "Refresh an access token",
		Tags:        []string{"Auth"},
	}, func(ctx context.Context, input *RefreshInput) (*RefreshOutput, error) {
		pair, err := authSvc.Refresh(ctx, input.Body.RefreshToken)
		if err != nil {
			if errors.Is(err, domain.ErrUnauthorized) {
				return nil, huma.Error401Unauthorized("invalid or expired refresh token")
			}
			return nil, huma.Error500InternalServerError("internal server error")
		}
		return &RefreshOutput{Body: pair}, nil
	})
}
