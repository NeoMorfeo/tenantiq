package i18n

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/validation"
	gi18n "github.com/nicksnyder/go-i18n/v2/i18n"
)

// --- Validation message overrides ---

// validationMessages maps English validation messages to i18n message IDs.
// These are the messages precomputed by Huma at schema init time.
var validationMessages = map[string]string{}

// OverrideHumaMessages replaces Huma's default validation messages with
// user-friendly English versions that we also have translated in our
// locale files. This must be called before registering Huma routes,
// since schemas are precomputed at registration time.
func OverrideHumaMessages() {
	// Replace the default messages with cleaner English.
	validation.MsgExpectedMinLength = "must be at least %d characters"
	validation.MsgExpectedMaxLength = "must be at most %d characters"
	validation.MsgExpectedMinimumNumber = "must be at least %v"
	validation.MsgExpectedMaximumNumber = "must be at most %v"
	validation.MsgExpectedRequiredProperty = "%s is required"
	validation.MsgExpectedOneOf = "must be one of: %s"
	validation.MsgExpectedRFC5322Email = "invalid email: %v"
	validation.MsgExpectedMatchPattern = "invalid format"
	validation.MsgExpectedMinItems = "must contain at least %d items"
	validation.MsgExpectedMaxItems = "must contain at most %d items"
	validation.MsgExpectedString = "must be a string"
	validation.MsgExpectedNumber = "must be a number"
	validation.MsgExpectedInteger = "must be an integer"
	validation.MsgExpectedBoolean = "must be a boolean"
	validation.MsgExpectedArray = "must be an array"
	validation.MsgExpectedObject = "must be an object"

	// Build a mapping from English messages to i18n keys for runtime translation.
	validationMessages = map[string]string{
		"must be a string":   "validation.must_be_string",
		"must be a number":   "validation.must_be_number",
		"must be an integer": "validation.must_be_integer",
		"must be a boolean":  "validation.must_be_boolean",
		"must be an array":   "validation.must_be_array",
		"must be an object":  "validation.must_be_object",
		"invalid format":     "validation.invalid_format",
		"validation failed":  "validation.failed",
	}
}

// Regex patterns for parametric validation messages.
var (
	reMinLength = regexp.MustCompile(`^must be at least (\d+) characters$`)
	reMaxLength = regexp.MustCompile(`^must be at most (\d+) characters$`)
	reMinNumber = regexp.MustCompile(`^must be at least (.+)$`)
	reMaxNumber = regexp.MustCompile(`^must be at most (.+)$`)
	reRequired  = regexp.MustCompile(`^(.+) is required$`)
	reOneOf     = regexp.MustCompile(`^must be one of: (.+)$`)
	reMinItems  = regexp.MustCompile(`^must contain at least (\d+) items$`)
	reMaxItems  = regexp.MustCompile(`^must contain at most (\d+) items$`)
	reEmail     = regexp.MustCompile(`^invalid email: (.+)$`)
)

// translateValidationMsg attempts to translate a validation message using the localizer.
func translateValidationMsg(l *Localizer_t, msg string) string {
	// Check static messages first.
	if id, ok := validationMessages[msg]; ok {
		if translated := T(l, id, nil); translated != id {
			return translated
		}
	}

	// Check parametric messages.
	if m := reMinLength.FindStringSubmatch(msg); m != nil {
		return T(l, "validation.min_length", map[string]string{"Min": m[1]})
	}
	if m := reMaxLength.FindStringSubmatch(msg); m != nil {
		return T(l, "validation.max_length", map[string]string{"Max": m[1]})
	}
	if m := reRequired.FindStringSubmatch(msg); m != nil {
		return T(l, "validation.required", map[string]string{"Field": m[1]})
	}
	if m := reOneOf.FindStringSubmatch(msg); m != nil {
		return T(l, "validation.one_of", map[string]string{"Values": m[1]})
	}
	if m := reMinItems.FindStringSubmatch(msg); m != nil {
		return T(l, "validation.min_items", map[string]string{"Min": m[1]})
	}
	if m := reMaxItems.FindStringSubmatch(msg); m != nil {
		return T(l, "validation.max_items", map[string]string{"Max": m[1]})
	}
	if m := reEmail.FindStringSubmatch(msg); m != nil {
		return T(l, "validation.invalid_email", map[string]string{"Detail": m[1]})
	}
	if m := reMinNumber.FindStringSubmatch(msg); m != nil {
		return T(l, "validation.min_number", map[string]string{"Min": m[1]})
	}
	if m := reMaxNumber.FindStringSubmatch(msg); m != nil {
		return T(l, "validation.max_number", map[string]string{"Max": m[1]})
	}

	// Fallback: return original.
	return msg
}

// Localizer_t is an alias for the go-i18n Localizer type.
type Localizer_t = gi18n.Localizer

// --- NewErrorWithContext override ---

// InstallErrorTranslator replaces Huma's NewErrorWithContext so that
// ALL error responses (handler errors + validation errors) are translated
// according to the Accept-Language header of each request.
func InstallErrorTranslator() {
	original := huma.NewError

	huma.NewErrorWithContext = func(ctx huma.Context, status int, msg string, errs ...error) huma.StatusError {
		l := Localizer(ctx.Context())

		// Translate the top-level detail message.
		translatedMsg := translateValidationMsg(l, msg)

		// Translate each error detail.
		translatedErrs := make([]error, len(errs))
		for i, err := range errs {
			var ed *huma.ErrorDetail
			if errors.As(err, &ed) {
				translatedErrs[i] = &huma.ErrorDetail{
					Message:  translateValidationMsg(l, ed.Message),
					Location: ed.Location,
					Value:    ed.Value,
				}
			} else {
				translatedErrs[i] = err
			}
		}

		result := original(status, translatedMsg, translatedErrs...)

		// Also translate the title (e.g. "Unprocessable Entity" → "Entidad no procesable").
		if em, ok := result.(*huma.ErrorModel); ok {
			if titleKey := fmt.Sprintf("http.%d", status); titleKey != "" {
				if translated := T(l, titleKey, nil); translated != titleKey {
					em.Title = translated
				}
			}
		}

		return result
	}
}

// --- Default ErrorFormatter override ---

// InstallErrorFormatter replaces Huma's ErrorFormatter so that messages
// precomputed by schemas can use cleaner English text.
func InstallErrorFormatter() {
	huma.ErrorFormatter = fmt.Sprintf
}

// SetupHuma configures all Huma i18n hooks. Call before registering routes.
func SetupHuma() {
	OverrideHumaMessages()
	InstallErrorTranslator()
}
