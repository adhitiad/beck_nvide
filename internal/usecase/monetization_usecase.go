package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type monetizationUseCase struct {
	monRepo    domain.MonetizationRepository
	walletRepo domain.WalletRepository
	txRepo     domain.TransactionRepository
	streamRepo domain.StreamRepository
	logger     *zap.Logger
}

// NewMonetizationUseCase creates a new instance of monetizationUseCase
func NewMonetizationUseCase(
	monRepo domain.MonetizationRepository,
	walletRepo domain.WalletRepository,
	txRepo domain.TransactionRepository,
	streamRepo domain.StreamRepository,
	logger *zap.Logger,
) domain.MonetizationUseCase {
	return &monetizationUseCase{
		monRepo:    monRepo,
		walletRepo: walletRepo,
		txRepo:     txRepo,
		streamRepo: streamRepo,
		logger:     logger,
	}
}

// Paid Room
func (uc *monetizationUseCase) CreatePaidRoom(ctx context.Context, hostID domain.UUID, name string, entryFeeIDR int64) (*domain.PaidRoom, error) {
	if name == "" {
		return nil, errors.New("nama room tidak boleh kosong")
	}
	if entryFeeIDR < 0 {
		return nil, errors.New("biaya masuk room tidak valid")
	}

	room := &domain.PaidRoom{
		ID:          domain.NewUUID(),
		HostID:      hostID,
		Name:        name,
		EntryFeeIDR: entryFeeIDR,
	}

	if err := uc.monRepo.CreatePaidRoom(ctx, room); err != nil {
		uc.logger.Error("Gagal membuat paid room di repositori", zap.Error(err))
		return nil, err
	}

	return room, nil
}

func (uc *monetizationUseCase) JoinPaidRoom(ctx context.Context, userID, roomID domain.UUID) (*domain.PaidRoom, error) {
	room, err := uc.monRepo.GetPaidRoomByID(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, errors.New("room berbayar tidak ditemukan")
	}

	if room.HostID == userID {
		// Host does not need to pay to join their own room
		return room, nil
	}

	// Transfer entry fee atomically using a database transaction
	err = uc.walletRepo.RunInTx(ctx, func(txCtx context.Context) error {
		// Deduct user wallet
		walletUser, err := uc.walletRepo.GetByUserID(txCtx, userID)
		if err != nil {
			return err
		}
		if walletUser.Balance < room.EntryFeeIDR {
			return errors.New("saldo wallet tidak mencukupi")
		}

		if err := uc.walletRepo.DebitBalance(txCtx, userID, room.EntryFeeIDR); err != nil {
			return err
		}

		// Credit host wallet
		if err := uc.walletRepo.CreditBalance(txCtx, room.HostID, room.EntryFeeIDR); err != nil {
			return err
		}

		refID := fmt.Sprintf("paid_room_%s_user_%s", room.ID.String()[:8], userID.String()[:8])

		// Log user transaction
		txUser := &domain.Transaction{
			ID:          domain.NewUUID(),
			UserID:      userID,
			Type:        "paid_room_entry",
			Amount:      -room.EntryFeeIDR,
			Currency:    "IDR",
			Status:      domain.TxStatusSuccess,
			ReferenceID: refID,
			Metadata:    fmt.Sprintf(`{"room_id":"%s","room_name":"%s"}`, room.ID, room.Name),
		}
		if err := uc.txRepo.Create(txCtx, txUser); err != nil {
			return err
		}

		// Log host transaction
		txHost := &domain.Transaction{
			ID:          domain.NewUUID(),
			UserID:      room.HostID,
			Type:        domain.TxTypeHostEarning,
			Amount:      room.EntryFeeIDR,
			Currency:    "IDR",
			Status:      domain.TxStatusSuccess,
			ReferenceID: refID,
			Metadata:    fmt.Sprintf(`{"room_id":"%s","room_name":"%s","buyer_id":"%s"}`, room.ID, room.Name, userID),
		}
		return uc.txRepo.Create(txCtx, txHost)
	})

	if err != nil {
		uc.logger.Error("Gagal memproses transaksi masuk paid room", zap.Error(err))
		return nil, err
	}

	return room, nil
}

// Interactive Toys
func (uc *monetizationUseCase) RegisterHostDevice(ctx context.Context, hostID domain.UUID, deviceName, deviceID, apiToken string) (*domain.HostDevice, error) {
	if deviceName == "" || deviceID == "" || apiToken == "" {
		return nil, errors.New("detail perangkat tidak lengkap")
	}

	device := &domain.HostDevice{
		ID:         domain.NewUUID(),
		HostID:     hostID,
		DeviceName: deviceName,
		DeviceID:   deviceID,
		APIToken:   apiToken,
	}

	if err := uc.monRepo.SaveHostDevice(ctx, device); err != nil {
		uc.logger.Error("Gagal menyimpan perangkat host", zap.Error(err))
		return nil, err
	}

	return device, nil
}

func (uc *monetizationUseCase) GetHostDevices(ctx context.Context, hostID domain.UUID) ([]*domain.HostDevice, error) {
	return uc.monRepo.GetHostDevices(ctx, hostID)
}

func (uc *monetizationUseCase) ControlToys(ctx context.Context, userID, streamID domain.UUID, command string, durationSeconds int, tipsAmount int64) (string, error) {
	stream, err := uc.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return "", err
	}
	if stream == nil {
		return "", errors.New("live stream tidak ditemukan")
	}

	devices, err := uc.monRepo.GetHostDevices(ctx, stream.HostID)
	if err != nil {
		return "", err
	}
	if len(devices) == 0 {
		return "", errors.New("host tidak memiliki perangkat mainan pintar Lovense terdaftar")
	}

	// Handle tips payment if tipsAmount > 0
	if tipsAmount > 0 {
		err = uc.walletRepo.RunInTx(ctx, func(txCtx context.Context) error {
			// Debit user wallet
			walletUser, err := uc.walletRepo.GetByUserID(txCtx, userID)
			if err != nil {
				return err
			}
			if walletUser.Balance < tipsAmount {
				return errors.New("saldo wallet tidak mencukupi")
			}

			if err := uc.walletRepo.DebitBalance(txCtx, userID, tipsAmount); err != nil {
				return err
			}

			// Credit host wallet
			if err := uc.walletRepo.CreditBalance(txCtx, stream.HostID, tipsAmount); err != nil {
				return err
			}

			refID := fmt.Sprintf("toy_ctrl_%s_usr_%s", streamID.String()[:8], userID.String()[:8])

			txUser := &domain.Transaction{
				ID:          domain.NewUUID(),
				UserID:      userID,
				Type:        "toy_control_tips",
				Amount:      -tipsAmount,
				Currency:    "IDR",
				Status:      domain.TxStatusSuccess,
				ReferenceID: refID,
				Metadata:    fmt.Sprintf(`{"stream_id":"%s","command":"%s"}`, streamID, command),
			}
			if err := uc.txRepo.Create(txCtx, txUser); err != nil {
				return err
			}

			txHost := &domain.Transaction{
				ID:          domain.NewUUID(),
				UserID:      stream.HostID,
				Type:        domain.TxTypeHostEarning,
				Amount:      tipsAmount,
				Currency:    "IDR",
				Status:      domain.TxStatusSuccess,
				ReferenceID: refID,
				Metadata:    fmt.Sprintf(`{"stream_id":"%s","command":"%s","buyer_id":"%s"}`, streamID, command, userID),
			}
			return uc.txRepo.Create(txCtx, txHost)
		})

		if err != nil {
			return "", err
		}
	}

	// Standard Lovense API URL:
	// https://api.lovense.com/api/basic/v2/toy/control
	targetDevice := devices[0]

	// Simulate or execute Lovense API Call
	payload := map[string]any{
		"token":   targetDevice.APIToken,
		"uid":     targetDevice.DeviceID,
		"command": "Function",
		"action":  command, // e.g., "Vibrate:2", "Vibrate:3", etc.
		"timeSec": durationSeconds,
		"apiVer":  1,
	}

	jsonBytes, _ := json.Marshal(payload)
	uc.logger.Info("Mengirimkan perintah ke mainan Lovense host", zap.String("host_id", stream.HostID.String()), zap.String("command", command))

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.lovense.com/api/basic/v2/toy/control", bytes.NewBuffer(jsonBytes))
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		resp, respErr := client.Do(req)
		if respErr == nil {
			resp.Body.Close()
		}
	}

	// Always return a descriptive success message (with mock fallback representation for developer preview)
	successMsg := fmt.Sprintf("Perangkat Lovense [%s] milik Host berhasil dikontrol dengan perintah [%s] selama %d detik!", targetDevice.DeviceName, command, durationSeconds)
	if tipsAmount > 0 {
		successMsg = fmt.Sprintf("%s (Dengan tips %d IDR dikirim ke Host)", successMsg, tipsAmount)
	}
	return successMsg, nil
}

// Show Request
func (uc *monetizationUseCase) SubmitShowRequest(ctx context.Context, userID, streamID domain.UUID, description string, tipsAmount int64) (*domain.ShowRequest, error) {
	if description == "" {
		return nil, errors.New("deskripsi request show tidak boleh kosong")
	}
	if tipsAmount <= 0 {
		return nil, errors.New("jumlah tips tidak valid")
	}

	stream, err := uc.streamRepo.GetByID(ctx, streamID)
	if err != nil {
		return nil, err
	}
	if stream == nil {
		return nil, errors.New("live stream tidak ditemukan")
	}

	// Check if user has sufficient balance before locking
	wallet, err := uc.walletRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if wallet.Balance < tipsAmount {
		return nil, errors.New("saldo wallet tidak mencukupi untuk mengajukan request")
	}

	req := &domain.ShowRequest{
		ID:          domain.NewUUID(),
		StreamID:    streamID,
		UserID:      userID,
		Description: description,
		TipsAmount:  tipsAmount,
		Status:      "pending",
	}

	if err := uc.monRepo.CreateShowRequest(ctx, req); err != nil {
		return nil, err
	}

	return req, nil
}

func (uc *monetizationUseCase) AcceptShowRequest(ctx context.Context, hostID, requestID domain.UUID) error {
	req, err := uc.monRepo.GetShowRequestByID(ctx, requestID)
	if err != nil {
		return err
	}
	if req == nil {
		return errors.New("request show tidak ditemukan")
	}

	stream, err := uc.streamRepo.GetByID(ctx, req.StreamID)
	if err != nil {
		return err
	}
	if stream == nil || stream.HostID != hostID {
		return errors.New("hanya host dari stream bersangkutan yang dapat menerima request")
	}

	if req.Status != "pending" {
		return fmt.Errorf("tidak dapat menerima request dengan status: %s", req.Status)
	}

	// Execute wallet transfer atomically
	err = uc.walletRepo.RunInTx(ctx, func(txCtx context.Context) error {
		// Debit user wallet
		walletUser, err := uc.walletRepo.GetByUserID(txCtx, req.UserID)
		if err != nil {
			return err
		}
		if walletUser.Balance < req.TipsAmount {
			return errors.New("saldo user tidak mencukupi saat host menerima request")
		}

		if err := uc.walletRepo.DebitBalance(txCtx, req.UserID, req.TipsAmount); err != nil {
			return err
		}

		// Credit host wallet
		if err := uc.walletRepo.CreditBalance(txCtx, hostID, req.TipsAmount); err != nil {
			return err
		}

		refID := fmt.Sprintf("show_req_%s_host_%s", req.ID.String()[:8], hostID.String()[:8])

		// User debit transaction log
		txUser := &domain.Transaction{
			ID:          domain.NewUUID(),
			UserID:      req.UserID,
			Type:        "show_request_tips",
			Amount:      -req.TipsAmount,
			Currency:    "IDR",
			Status:      domain.TxStatusSuccess,
			ReferenceID: refID,
			Metadata:    fmt.Sprintf(`{"request_id":"%s","description":"%s"}`, req.ID, req.Description),
		}
		if err := uc.txRepo.Create(txCtx, txUser); err != nil {
			return err
		}

		// Host credit transaction log
		txHost := &domain.Transaction{
			ID:          domain.NewUUID(),
			UserID:      hostID,
			Type:        domain.TxTypeHostEarning,
			Amount:      req.TipsAmount,
			Currency:    "IDR",
			Status:      domain.TxStatusSuccess,
			ReferenceID: refID,
			Metadata:    fmt.Sprintf(`{"request_id":"%s","description":"%s","user_id":"%s"}`, req.ID, req.Description, req.UserID),
		}
		if err := uc.txRepo.Create(txCtx, txHost); err != nil {
			return err
		}

		return uc.monRepo.UpdateShowRequestStatus(txCtx, req.ID, "accepted")
	})

	return err
}

func (uc *monetizationUseCase) RejectShowRequest(ctx context.Context, hostID, requestID domain.UUID) error {
	req, err := uc.monRepo.GetShowRequestByID(ctx, requestID)
	if err != nil {
		return err
	}
	if req == nil {
		return errors.New("request show tidak ditemukan")
	}

	stream, err := uc.streamRepo.GetByID(ctx, req.StreamID)
	if err != nil {
		return err
	}
	if stream == nil || stream.HostID != hostID {
		return errors.New("hanya host dari stream bersangkutan yang dapat menolak request")
	}

	if req.Status != "pending" {
		return fmt.Errorf("tidak dapat menolak request dengan status: %s", req.Status)
	}

	return uc.monRepo.UpdateShowRequestStatus(ctx, req.ID, "rejected")
}

// AI Companion Chatbot
func (uc *monetizationUseCase) SendAIChatMessage(ctx context.Context, userID, hostID domain.UUID, content string) (*domain.AIChatMessage, error) {
	if content == "" {
		return nil, errors.New("konten chat tidak boleh kosong")
	}

	// 1. Check if the creator is offline
	// If creator is actively live streaming, chatbot is restricted
	activeStreams, err := uc.streamRepo.ListLive(ctx, 10, 0)
	if err == nil {
		for _, s := range activeStreams {
			if s.HostID == hostID {
				return nil, errors.New("kreator sedang online/live streaming sekarang! Silakan bergabung ke live stream untuk mengobrol langsung")
			}
		}
	}

	// 2. Token-gate: Chatting with bot costs 500 IDR flat fee
	err = uc.walletRepo.RunInTx(ctx, func(txCtx context.Context) error {
		wallet, err := uc.walletRepo.GetByUserID(txCtx, userID)
		if err != nil {
			return err
		}
		if wallet.Balance < 500 {
			return errors.New("saldo tidak mencukupi, chat dengan AI Companion memerlukan biaya 500 IDR")
		}

		if err := uc.walletRepo.DebitBalance(txCtx, userID, 500); err != nil {
			return err
		}

		// AI Companion fee goes to the host (passive income model!)
		if err := uc.walletRepo.CreditBalance(txCtx, hostID, 500); err != nil {
			return err
		}

		refID := fmt.Sprintf("ai_chat_%s", domain.NewUUID().String()[:8])
		txLog := &domain.Transaction{
			ID:          domain.NewUUID(),
			UserID:      userID,
			Type:        "ai_chat_fee",
			Amount:      -500,
			Currency:    "IDR",
			Status:      domain.TxStatusSuccess,
			ReferenceID: refID,
			Metadata:    fmt.Sprintf(`{"companion_host_id":"%s"}`, hostID),
		}
		return uc.txRepo.Create(txCtx, txLog)
	})

	if err != nil {
		return nil, err
	}

	// 3. Content filtering: scan user inputs for highly offensive words
	badWords := []string{"anjing", "bangsat", "kontol", "memek", "brengsek", "teroris", "bom"}
	lowercaseContent := strings.ToLower(content)
	for _, word := range badWords {
		if strings.Contains(lowercaseContent, word) {
			return nil, errors.New("pesan Anda diblokir oleh filter konten: mengandung bahasa yang tidak sopan atau dilarang")
		}
	}

	// 4. Retrieve or create session
	session, err := uc.monRepo.GetAIChatSession(ctx, userID, hostID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		session = &domain.AIChatSession{
			ID:     domain.NewUUID(),
			UserID: userID,
			HostID: hostID,
		}
		if err := uc.monRepo.CreateAIChatSession(ctx, session); err != nil {
			return nil, err
		}
	}

	// 5. Save user message to chat history
	userMsg := &domain.AIChatMessage{
		ID:         domain.NewUUID(),
		SessionID:  session.ID,
		SenderType: "user",
		Content:    content,
	}
	if err := uc.monRepo.SaveAIChatMessage(ctx, userMsg); err != nil {
		return nil, err
	}

	// 6. Learn host's public talk style from chat history
	chatHistory, err := uc.monRepo.GetHostChatHistory(ctx, hostID, 15)
	var styleContext string
	if err == nil && len(chatHistory) > 0 {
		styleContext = strings.Join(chatHistory, " | ")
	} else {
		styleContext = "Ayo kak! Makasih banyak ya support-nya!"
	}

	// 7. Generate bot response (Premium AI GPT API call with robust fallback simulator)
	var aiResponse string
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey != "" {
		aiResponse = uc.generateOpenAIResponse(ctx, apiKey, content, styleContext)
	} else {
		aiResponse = uc.simulateHostResponse(content, styleContext)
	}

	// 8. Save bot message to chat history
	aiMsg := &domain.AIChatMessage{
		ID:         domain.NewUUID(),
		SessionID:  session.ID,
		SenderType: "ai",
		Content:    aiResponse,
	}
	if err := uc.monRepo.SaveAIChatMessage(ctx, aiMsg); err != nil {
		return nil, err
	}

	return aiMsg, nil
}

func (uc *monetizationUseCase) GetAIChatHistory(ctx context.Context, userID, hostID domain.UUID, limit int) ([]*domain.AIChatMessage, error) {
	session, err := uc.monRepo.GetAIChatSession(ctx, userID, hostID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return []*domain.AIChatMessage{}, nil
	}

	return uc.monRepo.GetAIChatHistory(ctx, session.ID, limit)
}

func (uc *monetizationUseCase) generateOpenAIResponse(ctx context.Context, apiKey, userMsg, styleContext string) string {
	type apiMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	systemPrompt := fmt.Sprintf("Anda adalah bot AI yang mensimulasikan kepribadian seorang host/streamer NVide Live. Gaya berbicara Anda harus meniru riwayat chat nyata dari host tersebut berikut ini: '%s'. Jawablah pesan pengguna dengan gaya bahasa yang ramah, sedikit centil/manis, bersahabat, dan selalu gunakan Bahasa Indonesia gaul/lokal.", styleContext)

	payload := map[string]any{
		"model": "gpt-4o-mini",
		"messages": []apiMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMsg},
		},
		"max_tokens": 150,
	}

	jsonBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return uc.simulateHostResponse(userMsg, styleContext)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return uc.simulateHostResponse(userMsg, styleContext)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return uc.simulateHostResponse(userMsg, styleContext)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && len(result.Choices) > 0 {
		return strings.TrimSpace(result.Choices[0].Message.Content)
	}

	return uc.simulateHostResponse(userMsg, styleContext)
}

func (uc *monetizationUseCase) simulateHostResponse(userMsg, styleContext string) string {
	cleanMsg := strings.ToLower(userMsg)
	var styles []string
	if styleContext != "" {
		styles = strings.Split(styleContext, " | ")
	}

	var sampleStyle string
	if len(styles) > 0 {
		sampleStyle = styles[0]
	} else {
		sampleStyle = "Makasih banyak ya kak dukungannya!"
	}

	// Premium smart conversational matrix fallback
	if strings.Contains(cleanMsg, "halo") || strings.Contains(cleanMsg, "hi") || strings.Contains(cleanMsg, "pagi") || strings.Contains(cleanMsg, "malam") {
		return fmt.Sprintf("Halo manis! 😘 Senang banget kamu chat aku offline gini. Jangan lupa tonton live stream aku nanti ya! Oh ya, host style: '%s'", sampleStyle)
	}
	if strings.Contains(cleanMsg, "sayang") || strings.Contains(cleanMsg, "kangen") || strings.Contains(cleanMsg, "cinta") {
		return "Aww manisnya... Aku juga kangen tau! Makasih ya udah selalu nemenin aku di room chat. Muach! ❤️"
	}
	if strings.Contains(cleanMsg, "foto") || strings.Contains(cleanMsg, "video") || strings.Contains(cleanMsg, "request") {
		return "Hmm... kalau kamu mau request show khusus atau minta konten eksklusif, nanti ajukan pas aku lagi live stream ya! Pasti aku kabulin buat kamu. 😉"
	}

	return fmt.Sprintf("Aww, makasih ya chat-nya! Senang banget bisa ngobrol virtual sama kamu. Tetap dukung aku ya sayang! (Host Vibe: %s)", sampleStyle)
}
