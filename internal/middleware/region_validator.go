package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"nvide-live/pkg/config"
)

// IsAllowedRegion check if the country is within allowed countries
func IsAllowedRegion(country string) bool {
	c := strings.ToLower(strings.TrimSpace(country))
	allowedRegions := config.Get().AllowedRegions
	if allowedRegions == "" {
		return false
	}
	allowed := strings.Split(allowedRegions, ",")
	for _, a := range allowed {
		if c == strings.ToLower(strings.TrimSpace(a)) {
			return true
		}
	}
	return false
}

// RegionValidator checks the country in the KYC submission body
func RegionValidator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only validate POST to /kyc/submit
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/kyc/submit") {
			next.ServeHTTP(w, r)
			return
		}

		// Read request body
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "INVALID_REQUEST", "Failed to read request body")
			return
		}

		// Restore body so next handlers can read it
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Decode partially to get country
		var submission struct {
			Country string `json:"country"`
		}

		if err := json.Unmarshal(bodyBytes, &submission); err != nil {
			writeJSONError(w, http.StatusBadRequest, "INVALID_JSON", "Failed to parse JSON body")
			return
		}

		// Check if country is valid
		if !IsAllowedRegion(submission.Country) {
			writeJSONError(w, http.StatusForbidden, "REGION_RESTRICTED", "KYC submissions are not accepted from your country/region")
			return
		}

		next.ServeHTTP(w, r)
	})
}
