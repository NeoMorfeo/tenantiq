package i18n_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/validation"
	"github.com/neomorfeo/tenantiq/internal/i18n"
)

// mockContext implements huma.Context with just enough to satisfy the interface.
type mockContext struct {
	ctx context.Context
}

func (m *mockContext) Operation() *huma.Operation                 { return nil }
func (m *mockContext) Context() context.Context                   { return m.ctx }
func (m *mockContext) TLS() *tls.ConnectionState                  { return nil }
func (m *mockContext) Version() huma.ProtoVersion                 { return huma.ProtoVersion{} }
func (m *mockContext) Method() string                             { return "GET" }
func (m *mockContext) Host() string                               { return "localhost" }
func (m *mockContext) RemoteAddr() string                         { return "127.0.0.1" }
func (m *mockContext) URL() url.URL                               { return url.URL{} }
func (m *mockContext) Param(string) string                        { return "" }
func (m *mockContext) Query(string) string                        { return "" }
func (m *mockContext) Header(string) string                       { return "" }
func (m *mockContext) EachHeader(func(string, string))            {}
func (m *mockContext) BodyReader() io.Reader                      { return nil }
func (m *mockContext) GetMultipartForm() (*multipart.Form, error) { return nil, nil }
func (m *mockContext) SetReadDeadline(time.Time) error            { return nil }
func (m *mockContext) SetStatus(int)                              {}
func (m *mockContext) Status() int                                { return 0 }
func (m *mockContext) SetHeader(string, string)                   {}
func (m *mockContext) AppendHeader(string, string)                {}
func (m *mockContext) BodyWriter() io.Writer                      { return nil }

func TestOverrideHumaMessages(t *testing.T) {
	i18n.OverrideHumaMessages()

	// Verify validation.Msg* variables are set to our custom messages.
	checks := []struct {
		name string
		got  string
		want string
	}{
		{"MsgExpectedMinLength", validation.MsgExpectedMinLength, "must be at least %d characters"},
		{"MsgExpectedMaxLength", validation.MsgExpectedMaxLength, "must be at most %d characters"},
		{"MsgExpectedMinimumNumber", validation.MsgExpectedMinimumNumber, "must be at least %v"},
		{"MsgExpectedMaximumNumber", validation.MsgExpectedMaximumNumber, "must be at most %v"},
		{"MsgExpectedRequiredProperty", validation.MsgExpectedRequiredProperty, "%s is required"},
		{"MsgExpectedOneOf", validation.MsgExpectedOneOf, "must be one of: %s"},
		{"MsgExpectedRFC5322Email", validation.MsgExpectedRFC5322Email, "invalid email: %v"},
		{"MsgExpectedMatchPattern", validation.MsgExpectedMatchPattern, "invalid format"},
		{"MsgExpectedMinItems", validation.MsgExpectedMinItems, "must contain at least %d items"},
		{"MsgExpectedMaxItems", validation.MsgExpectedMaxItems, "must contain at most %d items"},
		{"MsgExpectedString", validation.MsgExpectedString, "must be a string"},
		{"MsgExpectedNumber", validation.MsgExpectedNumber, "must be a number"},
		{"MsgExpectedInteger", validation.MsgExpectedInteger, "must be an integer"},
		{"MsgExpectedBoolean", validation.MsgExpectedBoolean, "must be a boolean"},
		{"MsgExpectedArray", validation.MsgExpectedArray, "must be an array"},
		{"MsgExpectedObject", validation.MsgExpectedObject, "must be an object"},
	}

	for _, tt := range checks {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}

	// Verify the validationMessages map is populated by checking that
	// translateValidationMsg works for static messages (it relies on the map).
	l := i18n.NewLocalizer("es")
	got := i18n.TranslateValidationMsg(l, "must be a string")
	if got != "debe ser una cadena de texto" {
		t.Errorf("validationMessages map not populated: TranslateValidationMsg returned %q", got)
	}
}

func TestTranslateValidationMsg(t *testing.T) {
	// Ensure the map is populated.
	i18n.OverrideHumaMessages()

	t.Run("static_english", func(t *testing.T) {
		l := i18n.NewLocalizer("en")

		tests := []struct {
			msg  string
			want string
		}{
			{"must be a string", "must be a string"},
			{"must be a number", "must be a number"},
			{"must be an integer", "must be an integer"},
			{"must be a boolean", "must be a boolean"},
			{"must be an array", "must be an array"},
			{"must be an object", "must be an object"},
			{"invalid format", "invalid format"},
			{"validation failed", "validation failed"},
		}

		for _, tt := range tests {
			t.Run(tt.msg, func(t *testing.T) {
				got := i18n.TranslateValidationMsg(l, tt.msg)
				if got != tt.want {
					t.Errorf("TranslateValidationMsg(%q) = %q, want %q", tt.msg, got, tt.want)
				}
			})
		}
	})

	t.Run("static_spanish", func(t *testing.T) {
		l := i18n.NewLocalizer("es")

		tests := []struct {
			msg  string
			want string
		}{
			{"must be a string", "debe ser una cadena de texto"},
			{"must be a number", "debe ser un n\u00famero"},
			{"must be an integer", "debe ser un entero"},
			{"must be a boolean", "debe ser un booleano"},
			{"must be an array", "debe ser un arreglo"},
			{"must be an object", "debe ser un objeto"},
			{"invalid format", "formato no v\u00e1lido"},
			{"validation failed", "la validaci\u00f3n fall\u00f3"},
		}

		for _, tt := range tests {
			t.Run(tt.msg, func(t *testing.T) {
				got := i18n.TranslateValidationMsg(l, tt.msg)
				if got != tt.want {
					t.Errorf("TranslateValidationMsg(%q) = %q, want %q", tt.msg, got, tt.want)
				}
			})
		}
	})

	t.Run("parametric_english", func(t *testing.T) {
		l := i18n.NewLocalizer("en")

		tests := []struct {
			name string
			msg  string
			want string
		}{
			{"min_length", "must be at least 3 characters", "must be at least 3 characters"},
			{"max_length", "must be at most 50 characters", "must be at most 50 characters"},
			{"required", "name is required", "name is required"},
			{"one_of", "must be one of: a, b, c", "must be one of: a, b, c"},
			{"min_items", "must contain at least 1 items", "must contain at least 1 items"},
			{"max_items", "must contain at most 10 items", "must contain at most 10 items"},
			{"email", "invalid email: bad@", "invalid email: bad@"},
			{"min_number", "must be at least 5", "must be at least 5"},
			{"max_number", "must be at most 100", "must be at most 100"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := i18n.TranslateValidationMsg(l, tt.msg)
				if got != tt.want {
					t.Errorf("TranslateValidationMsg(%q) = %q, want %q", tt.msg, got, tt.want)
				}
			})
		}
	})

	t.Run("parametric_spanish", func(t *testing.T) {
		l := i18n.NewLocalizer("es")

		tests := []struct {
			name string
			msg  string
			want string
		}{
			{"min_length", "must be at least 3 characters", "debe tener al menos 3 caracteres"},
			{"max_length", "must be at most 50 characters", "debe tener como m\u00e1ximo 50 caracteres"},
			{"required", "name is required", "name es obligatorio"},
			{"one_of", "must be one of: a, b, c", "debe ser uno de: a, b, c"},
			{"min_items", "must contain at least 1 items", "debe contener al menos 1 elementos"},
			{"max_items", "must contain at most 10 items", "debe contener como m\u00e1ximo 10 elementos"},
			{"email", "invalid email: bad@", "email no v\u00e1lido: bad@"},
			{"min_number", "must be at least 5", "debe ser al menos 5"},
			{"max_number", "must be at most 100", "debe ser como m\u00e1ximo 100"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := i18n.TranslateValidationMsg(l, tt.msg)
				if got != tt.want {
					t.Errorf("TranslateValidationMsg(%q) = %q, want %q", tt.msg, got, tt.want)
				}
			})
		}
	})

	t.Run("unknown_fallback", func(t *testing.T) {
		l := i18n.NewLocalizer("es")
		msg := "some completely unknown validation message"
		got := i18n.TranslateValidationMsg(l, msg)
		if got != msg {
			t.Errorf("expected fallback to original %q, got %q", msg, got)
		}
	})
}

func TestInstallErrorTranslator(t *testing.T) {
	i18n.OverrideHumaMessages()
	i18n.InstallErrorTranslator()

	if huma.NewErrorWithContext == nil {
		t.Fatal("expected NewErrorWithContext to be set, got nil")
	}

	// Create a context with Spanish localizer.
	ctx := i18n.WithLocalizer(context.Background(), i18n.NewLocalizer("es"))
	mc := &mockContext{ctx: ctx}

	// Call with a validation error message and error details.
	result := huma.NewErrorWithContext(mc, http.StatusUnprocessableEntity, "validation failed",
		&huma.ErrorDetail{
			Message:  "must be a string",
			Location: "body.name",
			Value:    123,
		},
		&huma.ErrorDetail{
			Message:  "must be at least 3 characters",
			Location: "body.slug",
			Value:    "ab",
		},
	)

	em, ok := result.(*huma.ErrorModel)
	if !ok {
		t.Fatalf("expected *huma.ErrorModel, got %T", result)
	}

	// Verify top-level detail is translated.
	if em.Detail != "la validaci\u00f3n fall\u00f3" {
		t.Errorf("Detail = %q, want %q", em.Detail, "la validaci\u00f3n fall\u00f3")
	}

	// Verify HTTP status title is translated.
	if em.Title != "Entidad no procesable" {
		t.Errorf("Title = %q, want %q", em.Title, "Entidad no procesable")
	}

	// Verify error details are translated.
	if len(em.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(em.Errors))
	}

	if em.Errors[0].Message != "debe ser una cadena de texto" {
		t.Errorf("Errors[0].Message = %q, want %q", em.Errors[0].Message, "debe ser una cadena de texto")
	}
	if em.Errors[0].Location != "body.name" {
		t.Errorf("Errors[0].Location = %q, want %q", em.Errors[0].Location, "body.name")
	}
	if em.Errors[0].Value != 123 {
		t.Errorf("Errors[0].Value = %v, want %v", em.Errors[0].Value, 123)
	}

	if em.Errors[1].Message != "debe tener al menos 3 caracteres" {
		t.Errorf("Errors[1].Message = %q, want %q", em.Errors[1].Message, "debe tener al menos 3 caracteres")
	}
	if em.Errors[1].Location != "body.slug" {
		t.Errorf("Errors[1].Location = %q, want %q", em.Errors[1].Location, "body.slug")
	}
}

func TestInstallErrorTranslator_NonErrorDetail(t *testing.T) {
	i18n.OverrideHumaMessages()
	i18n.InstallErrorTranslator()

	ctx := i18n.WithLocalizer(context.Background(), i18n.NewLocalizer("es"))
	mc := &mockContext{ctx: ctx}

	// Pass a plain error (not *huma.ErrorDetail) to ensure it passes through.
	plainErr := fmt.Errorf("some plain error")
	result := huma.NewErrorWithContext(mc, http.StatusBadRequest, "validation failed", plainErr)

	em, ok := result.(*huma.ErrorModel)
	if !ok {
		t.Fatalf("expected *huma.ErrorModel, got %T", result)
	}

	// The plain error should be preserved (not translated as ErrorDetail).
	if len(em.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(em.Errors))
	}
	if em.Errors[0].Message != "some plain error" {
		t.Errorf("Errors[0].Message = %q, want %q", em.Errors[0].Message, "some plain error")
	}
}

func TestInstallErrorTranslator_EnglishFallback(t *testing.T) {
	i18n.OverrideHumaMessages()
	i18n.InstallErrorTranslator()

	// Context without explicit localizer should default to English.
	mc := &mockContext{ctx: context.Background()}

	result := huma.NewErrorWithContext(mc, http.StatusUnprocessableEntity, "must be a string")

	em, ok := result.(*huma.ErrorModel)
	if !ok {
		t.Fatalf("expected *huma.ErrorModel, got %T", result)
	}

	// English messages should pass through unchanged.
	if em.Detail != "must be a string" {
		t.Errorf("Detail = %q, want %q", em.Detail, "must be a string")
	}
	if em.Title != "Unprocessable Entity" {
		t.Errorf("Title = %q, want %q", em.Title, "Unprocessable Entity")
	}
}

func TestInstallErrorFormatter(t *testing.T) {
	i18n.InstallErrorFormatter()

	if huma.ErrorFormatter == nil {
		t.Fatal("expected ErrorFormatter to be set, got nil")
	}

	// Verify it behaves like fmt.Sprintf.
	got := huma.ErrorFormatter("hello %s %d", "world", 42)
	want := "hello world 42"
	if got != want {
		t.Errorf("ErrorFormatter(%q) = %q, want %q", "hello %s %d", got, want)
	}
}

func TestSetupHuma(t *testing.T) {
	i18n.SetupHuma()

	// Verify OverrideHumaMessages was called: check a message variable.
	if validation.MsgExpectedString != "must be a string" {
		t.Errorf("MsgExpectedString = %q, want %q", validation.MsgExpectedString, "must be a string")
	}

	// Verify InstallErrorTranslator was called.
	if huma.NewErrorWithContext == nil {
		t.Fatal("expected NewErrorWithContext to be set after SetupHuma")
	}

	// Verify translation works end-to-end via SetupHuma.
	ctx := i18n.WithLocalizer(context.Background(), i18n.NewLocalizer("es"))
	mc := &mockContext{ctx: ctx}

	result := huma.NewErrorWithContext(mc, http.StatusNotFound, "validation failed")
	em, ok := result.(*huma.ErrorModel)
	if !ok {
		t.Fatalf("expected *huma.ErrorModel, got %T", result)
	}
	if em.Title != "No encontrado" {
		t.Errorf("Title = %q, want %q", em.Title, "No encontrado")
	}
}
