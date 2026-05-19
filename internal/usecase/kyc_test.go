package usecase

import (
	"os"
	"testing"

	"nvide-live/pkg/config"
)

func TestIsAllowedCountry(t *testing.T) {
	// Setup custom allowed regions via environment
	os.Setenv("ALLOWED_REGIONS", "indonesia,vietnam,thailand")
	defer os.Unsetenv("ALLOWED_REGIONS")

	// Reload config to pick up the env var
	config.Load()

	tests := []struct {
		country  string
		expected bool
	}{
		{"indonesia", true},
		{"Indonesia", true},
		{" INDONESIA  ", true},
		{"vietnam", true},
		{"thailand", true},
		{"malaysia", false},
		{"usa", false},
	}

	for _, tc := range tests {
		t.Run(tc.country, func(t *testing.T) {
			res := IsAllowedCountry(tc.country)
			if res != tc.expected {
				t.Errorf("IsAllowedCountry(%q) = %v; expected %v", tc.country, res, tc.expected)
			}
		})
	}
}

func TestIsLGBTIndicated(t *testing.T) {
	tests := []struct {
		name        string
		gender      string
		fullName    string
		documentURL string
		expected    bool
	}{
		{"Normal Male", "male", "Joko Susilo", "https://supabase/ktp1.jpg", false},
		{"Normal Female", "female", "Siti Aminah", "https://supabase/ktp2.jpg", false},
		{"Normal Pria", "pria", "Budi Santoso", "https://supabase/ktp3.jpg", false},
		{"LGBT Keyword in Gender", "transgender", "Joko", "https://supabase/ktp.jpg", true},
		{"LGBT Keyword in Name", "female", "Joko Queer", "https://supabase/ktp.jpg", true},
		{"LGBT Keyword in URL", "female", "Joko", "https://supabase/lgbt_card.jpg", true},
		{"Non-Binary Gender Blocked", "non-binary", "Joko", "https://supabase/ktp.jpg", true},
		{"Invalid Gender", "other", "Joko", "https://supabase/ktp.jpg", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := IsLGBTIndicated(tc.gender, tc.fullName, tc.documentURL)
			if res != tc.expected {
				t.Errorf("IsLGBTIndicated(%q, %q, %q) = %v; expected %v", tc.gender, tc.fullName, tc.documentURL, res, tc.expected)
			}
		})
	}
}
