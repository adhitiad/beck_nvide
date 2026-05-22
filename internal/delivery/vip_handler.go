package delivery

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"nvide-live/internal/domain"
	"nvide-live/internal/usecase"
)

type VIPHandler struct {
	vipUC *usecase.VIPUseCase
}

func NewVIPHandler(vipUC *usecase.VIPUseCase) *VIPHandler {
	return &VIPHandler{vipUC: vipUC}
}

func (h *VIPHandler) ListPlans(w http.ResponseWriter, r *http.Request) {
	levels, err := h.vipUC.ListPlans(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, levels)
}

func (h *VIPHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		Level string `json:"level"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	uv, err := h.vipUC.Subscribe(r.Context(), userID, req.Level)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, uv)
}

func (h *VIPHandler) GetMyVIP(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	uv, err := h.vipUC.GetMyVIP(r.Context(), userID)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{"vip": nil})
		return
	}
	respondJSON(w, http.StatusOK, uv)
}

func (h *VIPHandler) SetAutoRenew(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		AutoRenew bool `json:"auto_renew"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.vipUC.SetAutoRenew(r.Context(), userID, req.AutoRenew); err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *VIPHandler) GetEmoticons(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	emoticons, err := h.vipUC.GetEmoticons(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, emoticons)
}

func (h *VIPHandler) GetEntryEffect(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	effect, err := h.vipUC.GetEntryEffect(r.Context(), userID)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{"effect": nil})
		return
	}
	respondJSON(w, http.StatusOK, effect)
}

// ---- Royal Family Handler ----

type RoyalFamilyHandler struct {
	familyUC *usecase.RoyalFamilyUseCase
}

func NewRoyalFamilyHandler(familyUC *usecase.RoyalFamilyUseCase) *RoyalFamilyHandler {
	return &RoyalFamilyHandler{familyUC: familyUC}
}

func (h *RoyalFamilyHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	family, err := h.familyUC.CreateFamily(r.Context(), userID, req.Name, req.Description)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, family)
}

func (h *RoyalFamilyHandler) Join(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	familyID := domain.UUID(mux.Vars(r)["id"])
	if err := h.familyUC.JoinFamily(r.Context(), userID, familyID); err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "joined"})
}

func (h *RoyalFamilyHandler) Leave(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if err := h.familyUC.LeaveFamily(r.Context(), userID); err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "left"})
}

func (h *RoyalFamilyHandler) Contribute(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		Amount int64 `json:"amount"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	contrib, err := h.familyUC.Contribute(r.Context(), userID, req.Amount)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, contrib)
}

func (h *RoyalFamilyHandler) GetFamily(w http.ResponseWriter, r *http.Request) {
	familyID := domain.UUID(mux.Vars(r)["id"])
	family, err := h.familyUC.GetFamily(r.Context(), familyID)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, family)
}

func (h *RoyalFamilyHandler) GetMyFamily(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	family, membership, err := h.familyUC.GetMyFamily(r.Context(), userID)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"family":     family,
		"membership": membership,
	})
}

func (h *RoyalFamilyHandler) GetMembers(w http.ResponseWriter, r *http.Request) {
	familyID := domain.UUID(mux.Vars(r)["id"])
	limit, offset := getPagination(r)
	members, err := h.familyUC.GetMembers(r.Context(), familyID, limit, offset)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, members)
}

func (h *RoyalFamilyHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	familyID := domain.UUID(mux.Vars(r)["id"])
	members, err := h.familyUC.GetContributionLeaderboard(r.Context(), familyID, 50)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, members)
}

func (h *RoyalFamilyHandler) GetTopFamilies(w http.ResponseWriter, r *http.Request) {
	families, err := h.familyUC.GetTopFamilies(r.Context(), 50)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, families)
}
