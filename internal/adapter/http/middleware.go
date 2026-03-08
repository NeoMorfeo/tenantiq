package http

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"mime/multipart"
	"net/url"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/neomorfeo/tenantiq/internal/domain"
)

type contextKey string

const claimsKey contextKey = "claims"

// ClaimsFromContext extracts the authenticated user's claims from the context.
func ClaimsFromContext(ctx context.Context) (domain.TokenClaims, bool) {
	claims, ok := ctx.Value(claimsKey).(domain.TokenClaims)
	return claims, ok
}

// authContext wraps a huma.Context to override the Go context,
// allowing us to inject TokenClaims for downstream handlers.
type authContext struct {
	parent      huma.Context
	overrideCtx context.Context
}

func (c *authContext) Operation() *huma.Operation             { return c.parent.Operation() }
func (c *authContext) Context() context.Context               { return c.overrideCtx }
func (c *authContext) TLS() *tls.ConnectionState              { return c.parent.TLS() }
func (c *authContext) Version() huma.ProtoVersion             { return c.parent.Version() }
func (c *authContext) Method() string                         { return c.parent.Method() }
func (c *authContext) Host() string                           { return c.parent.Host() }
func (c *authContext) RemoteAddr() string                     { return c.parent.RemoteAddr() }
func (c *authContext) URL() url.URL                           { return c.parent.URL() }
func (c *authContext) Param(name string) string               { return c.parent.Param(name) }
func (c *authContext) Query(name string) string               { return c.parent.Query(name) }
func (c *authContext) Header(name string) string              { return c.parent.Header(name) }
func (c *authContext) EachHeader(cb func(name, value string)) { c.parent.EachHeader(cb) }
func (c *authContext) BodyReader() io.Reader                  { return c.parent.BodyReader() }
func (c *authContext) GetMultipartForm() (*multipart.Form, error) {
	return c.parent.GetMultipartForm()
}
func (c *authContext) SetReadDeadline(d time.Time) error { return c.parent.SetReadDeadline(d) }
func (c *authContext) SetStatus(code int)                { c.parent.SetStatus(code) }
func (c *authContext) Status() int                       { return c.parent.Status() }
func (c *authContext) SetHeader(name, value string)      { c.parent.SetHeader(name, value) }
func (c *authContext) AppendHeader(name, value string)   { c.parent.AppendHeader(name, value) }
func (c *authContext) BodyWriter() io.Writer             { return c.parent.BodyWriter() }

// NewHumaAuthMiddleware returns a Huma middleware that:
//   - Skips auth for operations without a Security requirement
//   - Validates the JWT and injects TokenClaims into context
//   - Checks role scopes if defined (e.g. {"bearer": ["superadmin"]})
func NewHumaAuthMiddleware(api huma.API, tokens domain.TokenService) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		op := ctx.Operation()

		// No security requirement → public endpoint.
		if len(op.Security) == 0 {
			next(ctx)
			return
		}

		// Extract Bearer token from Authorization header.
		header := ctx.Header("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			slog.WarnContext(ctx.Context(), "auth: missing or invalid header", "operation", op.OperationID)
			_ = huma.WriteErr(api, ctx, 401, "missing or invalid authorization header")
			return
		}

		token := strings.TrimPrefix(header, "Bearer ")

		claims, err := tokens.ValidateAccess(token)
		if err != nil {
			slog.WarnContext(ctx.Context(), "auth: invalid or expired token", "operation", op.OperationID)
			_ = huma.WriteErr(api, ctx, 401, "invalid or expired token")
			return
		}

		// Check role scopes if defined. Security is a list of maps,
		// e.g. [{"bearer": ["superadmin", "admin"]}].
		// If scopes are present, the user must have one of them.
		for _, req := range op.Security {
			if scopes, ok := req["bearer"]; ok && len(scopes) > 0 {
				allowed := false
				for _, scope := range scopes {
					if string(claims.Role) == scope {
						allowed = true
						break
					}
				}
				if !allowed {
					slog.WarnContext(ctx.Context(), "auth: insufficient permissions",
						"operation", op.OperationID, "role", string(claims.Role), "user_id", claims.UserID)
					_ = huma.WriteErr(api, ctx, 403, "insufficient permissions")
					return
				}
			}
		}

		// Inject claims into context for handlers.
		goCtx := context.WithValue(ctx.Context(), claimsKey, claims)
		next(&authContext{parent: ctx, overrideCtx: goCtx})
	}
}

// BearerSecurity is a convenience for operations requiring any authenticated user.
var BearerSecurity = []map[string][]string{{"bearer": {}}}

// RoleSecurity returns a Security requirement for specific roles.
func RoleSecurity(roles ...domain.Role) []map[string][]string {
	scopes := make([]string, len(roles))
	for i, r := range roles {
		scopes[i] = string(r)
	}
	return []map[string][]string{{"bearer": scopes}}
}
