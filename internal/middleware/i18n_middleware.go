package middleware

import (
	"context"
	"net/http"

	"nvide-live/pkg/i18n"
)

type I18nContextKey string

const (
	LangKey I18nContextKey = "lang"
)

// I18nMiddleware detects user language preferences automatically
type I18nMiddleware struct{}

// NewI18nMiddleware creates new i18n middleware
func NewI18nMiddleware() *I18nMiddleware {
	return &I18nMiddleware{}
}

// Middleware handles language parsing and injects lang code into context
func (m *I18nMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Check query parameter `lang`
		lang := r.URL.Query().Get("lang")

		// 2. If empty, check `Accept-Language` header
		if lang == "" {
			lang = r.Header.Get("Accept-Language")
		}

		translator := i18n.GetTranslator()
		
		var matchedLang string
		if r.URL.Query().Get("lang") != "" {
			matchedLang = translator.NormalizeLang(lang)
		} else {
			matchedLang = translator.ParseAcceptLanguage(lang)
		}

		// 3. Inject matched language into context
		ctx := context.WithValue(r.Context(), LangKey, matchedLang)

		// 4. Proceed
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetLangFromContext retrieves language from context, falling back to "en"
func GetLangFromContext(ctx context.Context) string {
	if ctx == nil {
		return "en"
	}
	val, ok := ctx.Value(LangKey).(string)
	if !ok || val == "" {
		return "en"
	}
	return val
}

// T translates a key based on the language found in context
func T(ctx context.Context, key string, args ...interface{}) string {
	lang := GetLangFromContext(ctx)
	return i18n.GetTranslator().T(lang, key, args...)
}
