package i18n

import (
	"testing"
)

func TestTranslator_NormalizeLang(t *testing.T) {
	tr := GetTranslator()

	tests := []struct {
		input    string
		expected string
	}{
		{"en", "en"},
		{"en-US", "en"},
		{"EN", "en"},
		{"id", "id"},
		{"id-ID", "id"},
		{"zh-CN", "zh"},
		{"ZH-TW", "zh"},
		{"pt-BR", "pt-br"},
		{"pt-br", "pt-br"},
		{"fil", "fil"},
		{"fil-PH", "fil"},
		{"th", "th"},
		{"ms", "ms"},
		{"my", "my"},
		{"km", "km"},
		{"es", "es"},
		{"vi", "vi"},
		{"unknown", "en"}, // Fallback
		{"", "en"},        // Fallback
	}

	for _, tt := range tests {
		actual := tr.NormalizeLang(tt.input)
		if actual != tt.expected {
			t.Errorf("NormalizeLang(%q) = %q; expected %q", tt.input, actual, tt.expected)
		}
	}
}

func TestTranslator_ParseAcceptLanguage(t *testing.T) {
	tr := GetTranslator()

	tests := []struct {
		input    string
		expected string
	}{
		{"id-ID,id;q=0.9,en-US;q=0.8,en;q=0.7", "id"},
		{"fil;q=0.9, en;q=0.8", "fil"},
		{"pt-BR,pt;q=0.9", "pt-br"},
		{"", "en"},
		{"fr-FR,fr;q=0.9", "en"}, // Unsupported language falls back to en
	}

	for _, tt := range tests {
		actual := tr.ParseAcceptLanguage(tt.input)
		if actual != tt.expected {
			t.Errorf("ParseAcceptLanguage(%q) = %q; expected %q", tt.input, actual, tt.expected)
		}
	}
}

func TestTranslator_Translate(t *testing.T) {
	tr := GetTranslator()

	// Seed some translations manually for test
	tr.SetTranslations("en", map[string]string{
		"unauthorized": "Unauthorized access. Please log in first.",
		"welcome":      "Welcome %s!",
	})
	tr.SetTranslations("id", map[string]string{
		"unauthorized": "Akses tidak sah. Silakan masuk terlebih dahulu.",
		"welcome":      "Selamat datang %s!",
	})

	// Test direct translations
	if res := tr.T("en", "unauthorized"); res != "Unauthorized access. Please log in first." {
		t.Errorf("Expected English unauthorized message, got %q", res)
	}

	if res := tr.T("id", "unauthorized"); res != "Akses tidak sah. Silakan masuk terlebih dahulu." {
		t.Errorf("Expected Indonesian unauthorized message, got %q", res)
	}

	// Test arguments
	if res := tr.T("en", "welcome", "John"); res != "Welcome John!" {
		t.Errorf("Expected 'Welcome John!', got %q", res)
	}

	if res := tr.T("id", "welcome", "John"); res != "Selamat datang John!" {
		t.Errorf("Expected 'Selamat datang John!', got %q", res)
	}

	// Test fallback
	if res := tr.T("fr", "unauthorized"); res != "Unauthorized access. Please log in first." {
		t.Errorf("Expected fallback English unauthorized message, got %q", res)
	}

	// Test key not found fallback
	if res := tr.T("en", "non_existent_key"); res != "non_existent_key" {
		t.Errorf("Expected key itself when not found, got %q", res)
	}
}
