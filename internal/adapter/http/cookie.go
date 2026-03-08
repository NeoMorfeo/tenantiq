package http

import (
	"context"
	"net/http"
	"time"

	"github.com/neomorfeo/tenantiq/internal/i18n"
)

type ctxKey string

const (
	writerKey  ctxKey = "http.ResponseWriter"
	requestKey ctxKey = "http.Request"
)

// InjectHTTP is a chi middleware that stores the http.ResponseWriter,
// *http.Request, and an i18n localizer (from Accept-Language) in the
// request context so Huma handlers can set cookies, read request
// cookies, and translate error messages.
func InjectHTTP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), writerKey, w)
		ctx = context.WithValue(ctx, requestKey, r)

		// Inject i18n localizer based on Accept-Language header.
		lang := r.Header.Get("Accept-Language")
		if lang == "" {
			lang = "en"
		}
		ctx = i18n.WithLocalizer(ctx, i18n.NewLocalizer(lang))

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// responseWriter extracts the http.ResponseWriter from context.
func responseWriter(ctx context.Context) http.ResponseWriter {
	w, _ := ctx.Value(writerKey).(http.ResponseWriter)
	return w
}

// cookieValue reads a named cookie from the request stored in context.
func cookieValue(ctx context.Context, name string) string {
	r, _ := ctx.Value(requestKey).(*http.Request)
	if r == nil {
		return ""
	}
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

// CookieConfig holds cookie security settings.
type CookieConfig struct {
	Secure     bool // true in production (HTTPS)
	Domain     string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

const (
	accessCookieName  = "tenantiq_access"
	refreshCookieName = "tenantiq_refresh"
)

// setAuthCookies writes HttpOnly cookies for access and refresh tokens.
func setAuthCookies(w http.ResponseWriter, accessToken, refreshToken string, cfg CookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Value:    accessToken,
		Path:     "/",
		Domain:   cfg.Domain,
		MaxAge:   int(cfg.AccessTTL.Seconds()),
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    refreshToken,
		Path:     "/api/v1/auth", // only sent to auth endpoints
		Domain:   cfg.Domain,
		MaxAge:   int(cfg.RefreshTTL.Seconds()),
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// clearAuthCookies removes auth cookies by setting MaxAge to -1.
func clearAuthCookies(w http.ResponseWriter, cfg CookieConfig) {
	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName,
		Value:    "",
		Path:     "/",
		Domain:   cfg.Domain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		Domain:   cfg.Domain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}
