package delivery

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"nvide-live/internal/domain"
	"nvide-live/internal/middleware"
)

// --- Standalone helper functions for platform overhaul handlers ---

// getUserID extracts the authenticated user ID from request context
func getUserID(r *http.Request) domain.UUID {
	userID, _ := middleware.GetUserIDFromContext(r.Context())
	return userID
}

// respondJSON writes a JSON response with the given status code
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError writes a JSON error response
func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{
		"error_code": http.StatusText(status),
		"message":    message,
	})
}

// handleDomainError maps domain error codes to HTTP status codes
func handleDomainError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	// Check for value type DomainError
	var domainErr domain.DomainError
	if errors.As(err, &domainErr) {
		switch domainErr.Code {
		case domain.ErrCodeNotFound:
			respondError(w, http.StatusNotFound, domainErr.Message)
		case domain.ErrCodeValidation:
			respondError(w, http.StatusBadRequest, domainErr.Message)
		case domain.ErrCodeConflict:
			respondError(w, http.StatusConflict, domainErr.Message)
		case domain.ErrCodeForbidden:
			respondError(w, http.StatusForbidden, domainErr.Message)
		case domain.ErrCodeUnauthorized:
			respondError(w, http.StatusUnauthorized, domainErr.Message)
		default:
			respondError(w, http.StatusInternalServerError, domainErr.Message)
		}
		return
	}

	respondError(w, http.StatusInternalServerError, err.Error())
}

// getPagination extracts limit and offset from query parameters with defaults
func getPagination(r *http.Request) (int, int) {
	limit := 20
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 {
			limit = v
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if v, err := strconv.Atoi(o); err == nil && v >= 0 {
			offset = v
		}
	}
	return limit, offset
}

// suppress unused import warning
var _ = strings.Contains
