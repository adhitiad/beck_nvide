package delivery

import (
	"encoding/json"
	"net/http"
	"strings"

	"nvide-live/internal/middleware"
	"nvide-live/internal/usecase"
)

// Register handles user registration
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Basic validation
	if req.Username == "" || req.Email == "" || req.Password == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Username, email, and password are required")
		return
	}

	user, err := h.authUseCase.Register(r.Context(), &usecase.RegisterRequest{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	response := &AuthResponse{
		User: toUserDTO(user),
	}

	h.writeJSON(w, http.StatusCreated, response)
}

// Login handles user login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	accessToken, refreshToken, user, err := h.authUseCase.Login(r.Context(), &usecase.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.handleError(w, err)
		return
	}

	response := &AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toUserDTO(user),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// Refresh handles token refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.RefreshToken == "" {
		h.writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Refresh token is required")
		return
	}

	newAccessToken, newRefreshToken, user, err := h.authUseCase.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		h.handleError(w, err)
		return
	}

	response := &AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		User:         toUserDTO(user),
	}

	h.writeJSON(w, http.StatusOK, response)
}

// Logout handles user logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get tokens from headers or body
	accessToken := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")

	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	// Try to decode body if present
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	if err := h.authUseCase.Logout(r.Context(), accessToken, req.RefreshToken); err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// Me returns current user profile
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context (set by auth middleware)
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "User not authenticated")
		return
	}

	user, err := h.userUseCase.GetProfile(r.Context(), userID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeJSON(w, http.StatusOK, toUserDTO(user))
}
