package delivery

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	"nvide-live/internal/domain"
	"nvide-live/internal/usecase"
)

// ---- Inventory Handler ----

type InventoryHandler struct {
	inventoryUC *usecase.InventoryUseCase
}

func NewInventoryHandler(inventoryUC *usecase.InventoryUseCase) *InventoryHandler {
	return &InventoryHandler{inventoryUC: inventoryUC}
}

func (h *InventoryHandler) GetMyInventory(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	limit, offset := getPagination(r)
	items, err := h.inventoryUC.GetMyInventory(r.Context(), userID, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, items)
}

func (h *InventoryHandler) UseItem(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	itemID := domain.UUID(mux.Vars(r)["itemId"])
	var req struct {
		Quantity int `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.inventoryUC.UseItem(r.Context(), userID, itemID, req.Quantity); err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "used"})
}

func (h *InventoryHandler) ListCatalog(w http.ResponseWriter, r *http.Request) {
	itemType := r.URL.Query().Get("type")
	items, err := h.inventoryUC.ListCatalog(r.Context(), itemType)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, items)
}

// ---- Wheel Handler ----

type WheelHandler struct {
	wheelUC *usecase.WheelUseCase
}

func NewWheelHandler(wheelUC *usecase.WheelUseCase) *WheelHandler {
	return &WheelHandler{wheelUC: wheelUC}
}

func (h *WheelHandler) GetPrizes(w http.ResponseWriter, r *http.Request) {
	prizes, err := h.wheelUC.GetPrizes(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, prizes)
}

func (h *WheelHandler) Spin(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	spin, err := h.wheelUC.Spin(r.Context(), userID)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, spin)
}

func (h *WheelHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	limit, offset := getPagination(r)
	spins, err := h.wheelUC.GetHistory(r.Context(), userID, limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, spins)
}

// ---- Mission Handler ----

type MissionHandler struct {
	missionUC *usecase.MissionUseCase
}

func NewMissionHandler(missionUC *usecase.MissionUseCase) *MissionHandler {
	return &MissionHandler{missionUC: missionUC}
}

func (h *MissionHandler) GetDailyMissions(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	missions, err := h.missionUC.GetDailyMissions(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, missions)
}

func (h *MissionHandler) ClaimReward(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	missionID := domain.UUID(mux.Vars(r)["missionId"])
	if err := h.missionUC.ClaimReward(r.Context(), userID, missionID); err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "claimed"})
}

func (h *MissionHandler) GetBadges(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	badges, err := h.missionUC.GetBadges(r.Context(), userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, badges)
}

func (h *MissionHandler) GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	lbType := r.URL.Query().Get("type")
	period := r.URL.Query().Get("period")
	if lbType == "" {
		lbType = domain.LeaderboardHostIncome
	}
	if period == "" {
		period = domain.PeriodDaily
	}
	entries, err := h.missionUC.GetLeaderboard(r.Context(), lbType, period, 50)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, entries)
}

// ---- Voice Room Handler ----

type VoiceRoomHandler struct {
	voiceUC *usecase.VoiceRoomUseCase
}

func NewVoiceRoomHandler(voiceUC *usecase.VoiceRoomUseCase) *VoiceRoomHandler {
	return &VoiceRoomHandler{voiceUC: voiceUC}
}

func (h *VoiceRoomHandler) CreateRoom(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		MaxSpeakers int    `json:"max_speakers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	room, err := h.voiceUC.CreateRoom(r.Context(), userID, req.Title, req.Description, req.MaxSpeakers)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, room)
}

func (h *VoiceRoomHandler) JoinRoom(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	roomID := domain.UUID(mux.Vars(r)["id"])
	if err := h.voiceUC.JoinRoom(r.Context(), userID, roomID); err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "joined"})
}

func (h *VoiceRoomHandler) LeaveRoom(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	roomID := domain.UUID(mux.Vars(r)["id"])
	if err := h.voiceUC.LeaveRoom(r.Context(), userID, roomID); err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "left"})
}

func (h *VoiceRoomHandler) RequestStage(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	roomID := domain.UUID(mux.Vars(r)["id"])
	if err := h.voiceUC.RequestStage(r.Context(), userID, roomID); err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "stage_requested"})
}

func (h *VoiceRoomHandler) EndRoom(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	roomID := domain.UUID(mux.Vars(r)["id"])
	if err := h.voiceUC.EndRoom(r.Context(), userID, roomID); err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ended"})
}

func (h *VoiceRoomHandler) ListActive(w http.ResponseWriter, r *http.Request) {
	limit, offset := getPagination(r)
	rooms, err := h.voiceUC.ListActive(r.Context(), limit, offset)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, rooms)
}

func (h *VoiceRoomHandler) GetRoom(w http.ResponseWriter, r *http.Request) {
	roomID := domain.UUID(mux.Vars(r)["id"])
	room, err := h.voiceUC.GetRoom(r.Context(), roomID)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, room)
}

// ---- Lucky Bag Handler ----

type LuckyBagHandler struct {
	luckyBagUC *usecase.LuckyBagUseCase
}

func NewLuckyBagHandler(luckyBagUC *usecase.LuckyBagUseCase) *LuckyBagHandler {
	return &LuckyBagHandler{luckyBagUC: luckyBagUC}
}

func (h *LuckyBagHandler) CreateBag(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	var req struct {
		StreamID   string `json:"stream_id"`
		MinValue   int64  `json:"min_value"`
		MaxValue   int64  `json:"max_value"`
		TotalCount int    `json:"total_count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	bag, err := h.luckyBagUC.CreateBag(r.Context(), userID, domain.UUID(req.StreamID),
		req.MinValue, req.MaxValue, req.TotalCount)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusCreated, bag)
}

func (h *LuckyBagHandler) ClaimBag(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	bagID := domain.UUID(mux.Vars(r)["id"])
	claim, err := h.luckyBagUC.ClaimBag(r.Context(), userID, bagID)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, claim)
}

func (h *LuckyBagHandler) GetActiveByStream(w http.ResponseWriter, r *http.Request) {
	streamID := domain.UUID(mux.Vars(r)["streamId"])
	bags, err := h.luckyBagUC.GetActiveByStream(r.Context(), streamID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, bags)
}

// ---- Host Level Handler ----

type HostLevelHandler struct {
	levelUC *usecase.HostLevelUseCase
}

func NewHostLevelHandler(levelUC *usecase.HostLevelUseCase) *HostLevelHandler {
	return &HostLevelHandler{levelUC: levelUC}
}

func (h *HostLevelHandler) GetLevels(w http.ResponseWriter, r *http.Request) {
	levels, err := h.levelUC.GetLevels(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, levels)
}

func (h *HostLevelHandler) GetMyLevel(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	level, err := h.levelUC.GetHostLevel(r.Context(), userID)
	if err != nil {
		handleDomainError(w, err)
		return
	}
	respondJSON(w, http.StatusOK, level)
}
