package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Go:embed locales/*.json
//go:embed all:locales
var embedFS embed.FS

type Translator struct {
	translations map[string]map[string]string // lang -> key -> translation
	fallbackLang string
	mu           sync.RWMutex
}

var (
	instance *Translator
	once     sync.Once
)

// GetTranslator returns the singleton instance of Translator
func GetTranslator() *Translator {
	once.Do(func() {
		instance = &Translator{
			translations: make(map[string]map[string]string),
			fallbackLang: "en",
		}
	})
	return instance
}

// LoadTranslations loads JSON translations from locales directory on disk or fallback embed
func (t *Translator) LoadTranslations(localesDir string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Clear existing
	t.translations = make(map[string]map[string]string)

	// List of standard languages we must support
	supportedLangs := []string{"en", "id", "zh", "fil", "th", "ms", "my", "km", "es", "vi", "pt-br"}

	// Try reading from disk first
	loadedFromDisk := false
	if localesDir != "" {
		if _, err := os.Stat(localesDir); err == nil {
			for _, lang := range supportedLangs {
				filePath := filepath.Join(localesDir, lang+".json")
				data, err := os.ReadFile(filePath)
				if err == nil {
					var dict map[string]string
					if err := json.Unmarshal(data, &dict); err == nil {
						t.translations[lang] = dict
						loadedFromDisk = true
					}
				}
			}
		}
	}

	// Fallback to embed FS if disk was not used or incomplete
	if !loadedFromDisk {
		for _, lang := range supportedLangs {
			// Embedded files reside in locales/ subdirectory or root of embedded fs depending on structural paths
			filePath := fmt.Sprintf("locales/%s.json", lang)
			data, err := embedFS.ReadFile(filePath)
			if err != nil {
				// Try without prefix just in case
				filePath = lang + ".json"
				data, err = embedFS.ReadFile(filePath)
			}

			if err == nil {
				var dict map[string]string
				if err := json.Unmarshal(data, &dict); err == nil {
					// Merge or set
					if _, exists := t.translations[lang]; !exists {
						t.translations[lang] = dict
					}
				}
			}
		}
	}

	// Always ensure fallback language (en) is populated with at least something to avoid panics
	if _, exists := t.translations["en"]; !exists {
		t.translations["en"] = map[string]string{
			"unauthorized":     "Unauthorized access",
			"forbidden":        "Access forbidden",
			"not_found":        "Resource not found",
			"internal_error":   "Internal server error",
			"validation_error": "Validation error",
			"invalid_request":  "Invalid request body",
		}
	}

	return nil
}

// SetTranslations manually sets translation dictionary (useful for tests)
func (t *Translator) SetTranslations(lang string, dict map[string]string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.translations[lang] = dict
}

// T translates a key into the given language with optional arguments
func (t *Translator) T(lang string, key string, args ...interface{}) string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Clean language code (e.g. en-US -> en)
	lang = t.NormalizeLang(lang)

	// Fetch target language map
	langMap, ok := t.translations[lang]
	if !ok {
		// Fallback to en
		langMap = t.translations[t.fallbackLang]
	}

	val, found := langMap[key]
	if !found {
		// Fallback to en translation of key if target was different
		if lang != t.fallbackLang {
			if fallbackMap, exists := t.translations[t.fallbackLang]; exists {
				val, found = fallbackMap[key]
			}
		}
		
		if !found {
			// If still not found, return key itself
			return key
		}
	}

	if len(args) > 0 {
		return fmt.Sprintf(val, args...)
	}
	return val
}

// NormalizeLang converts strings like "en-US", "EN", "pt-BR" to "en", "en", "pt-br"
func (t *Translator) NormalizeLang(lang string) string {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		return t.fallbackLang
	}

	// Split by hyphens or underscores (e.g., zh-CN -> zh, pt-BR -> pt-br)
	parts := strings.FieldsFunc(lang, func(r rune) bool {
		return r == '-' || r == '_'
	})

	if len(parts) == 0 {
		return t.fallbackLang
	}

	base := parts[0]

	// Special case for pt-br since user specified pt-br as standard
	if base == "pt" && len(parts) > 1 && parts[1] == "br" {
		return "pt-br"
	}

	// Check if this normalized base is supported
	supported := map[string]bool{
		"en":    true,
		"id":    true,
		"zh":    true,
		"fil":   true,
		"th":    true,
		"ms":    true,
		"my":    true,
		"km":    true,
		"es":    true,
		"vi":    true,
		"pt-br": true,
	}

	if supported[base] {
		return base
	}

	// Try checking the full original string (without region split, e.g. pt-br)
	fullCleaned := strings.Join(parts, "-")
	if supported[fullCleaned] {
		return fullCleaned
	}

	return t.fallbackLang
}

// ParseAcceptLanguage parses Accept-Language header and returns the best matched supported language
func (t *Translator) ParseAcceptLanguage(header string) string {
	if header == "" {
		return t.fallbackLang
	}

	// E.g. Accept-Language: id-ID,id;q=0.9,en-US;q=0.8,en;q=0.7
	segments := strings.Split(header, ",")
	for _, seg := range segments {
		// Clean parameters like ;q=0.9
		parts := strings.Split(seg, ";")
		langToken := strings.TrimSpace(parts[0])
		
		norm := t.NormalizeLang(langToken)
		if norm != t.fallbackLang {
			return norm
		}
	}

	return t.fallbackLang
}

// EmbedLocales is a helper to verify embedded fs is working
func (t *Translator) EmbedLocales() fs.FS {
	return embedFS
}
