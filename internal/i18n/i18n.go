// Package i18n provides message translation using go-i18n.
// Translation files are embedded and loaded at init time.
package i18n

import (
	"context"
	"embed"
	"encoding/json"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.json
var localeFS embed.FS

// Bundle holds all loaded translations.
var Bundle *i18n.Bundle

func init() {
	Bundle = i18n.NewBundle(language.English)
	Bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// Load all locale files from the embedded filesystem.
	for _, name := range []string{"locales/en.json", "locales/es.json"} {
		data, err := localeFS.ReadFile(name)
		if err != nil {
			panic("i18n: failed to read " + name + ": " + err.Error())
		}
		Bundle.MustParseMessageFileBytes(data, name)
	}
}

// NewLocalizer creates a localizer for the given language tags.
func NewLocalizer(langs ...string) *i18n.Localizer {
	return i18n.NewLocalizer(Bundle, langs...)
}

// T translates a message ID using the given localizer.
// If templateData is nil, no interpolation is performed.
func T(l *i18n.Localizer, id string, templateData map[string]string) string {
	msg, err := l.Localize(&i18n.LocalizeConfig{
		MessageID:    id,
		TemplateData: templateData,
	})
	if err != nil {
		// Fallback: return the message ID itself so we never return empty.
		return id
	}
	return msg
}

// --- Context integration ---

type ctxKey string

const localizerKey ctxKey = "i18n_localizer"

// WithLocalizer stores a localizer in the context.
func WithLocalizer(ctx context.Context, l *i18n.Localizer) context.Context {
	return context.WithValue(ctx, localizerKey, l)
}

// Localizer retrieves the localizer from the context.
// Falls back to English if none is set.
func Localizer(ctx context.Context) *i18n.Localizer {
	if l, ok := ctx.Value(localizerKey).(*i18n.Localizer); ok {
		return l
	}
	return NewLocalizer("en")
}
