package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"nvide-live/pkg/config"
)

func TestRegionValidator(t *testing.T) {
	// Setup custom allowed regions via environment
	os.Setenv("ALLOWED_REGIONS", "indonesia,vietnam,thailand")
	defer os.Unsetenv("ALLOWED_REGIONS")

	// Reload config to pick up the env var
	config.Load()

	// Create a dummy next handler that returns 200 OK
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handlerToTest := RegionValidator(nextHandler)

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{
			"Allowed Country (Indonesia)",
			http.MethodPost,
			"/api/v1/kyc/submit",
			`{"country": "indonesia"}`,
			http.StatusOK,
		},
		{
			"Allowed Country Mixed Case",
			http.MethodPost,
			"/api/v1/kyc/submit",
			`{"country": "ViEtNaM"}`,
			http.StatusOK,
		},
		{
			"Blocked Country (USA)",
			http.MethodPost,
			"/api/v1/kyc/submit",
			`{"country": "usa"}`,
			http.StatusForbidden,
		},
		{
			"Non-matching Endpoint Allowed Through",
			http.MethodPost,
			"/api/v1/user/profile",
			`{"country": "usa"}`,
			http.StatusOK,
		},
		{
			"Non-matching Method Allowed Through",
			http.MethodGet,
			"/api/v1/kyc/submit",
			`{"country": "usa"}`,
			http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			handlerToTest.ServeHTTP(rr, req)

			if rr.Code != tc.expectedStatus {
				t.Errorf("expected status %d, got %d. Response: %s", tc.expectedStatus, rr.Code, rr.Body.String())
			}
		})
	}
}
