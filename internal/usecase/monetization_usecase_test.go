package usecase

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

// Mock MonetizationRepository
type mockMonRepo struct {
	CreatePaidRoomFunc          func(ctx context.Context, room *domain.PaidRoom) error
	GetPaidRoomByIDFunc         func(ctx context.Context, id domain.UUID) (*domain.PaidRoom, error)
	ListPaidRoomsByHostFunc     func(ctx context.Context, hostID domain.UUID) ([]*domain.PaidRoom, error)
	SaveHostDeviceFunc          func(ctx context.Context, device *domain.HostDevice) error
	GetHostDevicesFunc          func(ctx context.Context, hostID domain.UUID) ([]*domain.HostDevice, error)
	CreateShowRequestFunc       func(ctx context.Context, req *domain.ShowRequest) error
	GetShowRequestByIDFunc      func(ctx context.Context, id domain.UUID) (*domain.ShowRequest, error)
	UpdateShowRequestStatusFunc func(ctx context.Context, id domain.UUID, status string) error
	CreateAIChatSessionFunc     func(ctx context.Context, sess *domain.AIChatSession) error
	GetAIChatSessionFunc        func(ctx context.Context, userID, hostID domain.UUID) (*domain.AIChatSession, error)
	SaveAIChatMessageFunc       func(ctx context.Context, msg *domain.AIChatMessage) error
	GetAIChatHistoryFunc        func(ctx context.Context, sessionID domain.UUID, limit int) ([]*domain.AIChatMessage, error)
	GetHostChatHistoryFunc      func(ctx context.Context, hostID domain.UUID, limit int) ([]string, error)
}

func (m *mockMonRepo) CreatePaidRoom(ctx context.Context, room *domain.PaidRoom) error {
	if m.CreatePaidRoomFunc != nil {
		return m.CreatePaidRoomFunc(ctx, room)
	}
	return nil
}
func (m *mockMonRepo) GetPaidRoomByID(ctx context.Context, id domain.UUID) (*domain.PaidRoom, error) {
	if m.GetPaidRoomByIDFunc != nil {
		return m.GetPaidRoomByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockMonRepo) ListPaidRoomsByHost(ctx context.Context, hostID domain.UUID) ([]*domain.PaidRoom, error) {
	if m.ListPaidRoomsByHostFunc != nil {
		return m.ListPaidRoomsByHostFunc(ctx, hostID)
	}
	return nil, nil
}
func (m *mockMonRepo) SaveHostDevice(ctx context.Context, device *domain.HostDevice) error {
	if m.SaveHostDeviceFunc != nil {
		return m.SaveHostDeviceFunc(ctx, device)
	}
	return nil
}
func (m *mockMonRepo) GetHostDevices(ctx context.Context, hostID domain.UUID) ([]*domain.HostDevice, error) {
	if m.GetHostDevicesFunc != nil {
		return m.GetHostDevicesFunc(ctx, hostID)
	}
	return nil, nil
}
func (m *mockMonRepo) CreateShowRequest(ctx context.Context, req *domain.ShowRequest) error {
	if m.CreateShowRequestFunc != nil {
		return m.CreateShowRequestFunc(ctx, req)
	}
	return nil
}
func (m *mockMonRepo) GetShowRequestByID(ctx context.Context, id domain.UUID) (*domain.ShowRequest, error) {
	if m.GetShowRequestByIDFunc != nil {
		return m.GetShowRequestByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockMonRepo) UpdateShowRequestStatus(ctx context.Context, id domain.UUID, status string) error {
	if m.UpdateShowRequestStatusFunc != nil {
		return m.UpdateShowRequestStatusFunc(ctx, id, status)
	}
	return nil
}
func (m *mockMonRepo) CreateAIChatSession(ctx context.Context, sess *domain.AIChatSession) error {
	if m.CreateAIChatSessionFunc != nil {
		return m.CreateAIChatSessionFunc(ctx, sess)
	}
	return nil
}
func (m *mockMonRepo) GetAIChatSession(ctx context.Context, userID, hostID domain.UUID) (*domain.AIChatSession, error) {
	if m.GetAIChatSessionFunc != nil {
		return m.GetAIChatSessionFunc(ctx, userID, hostID)
	}
	return nil, nil
}
func (m *mockMonRepo) SaveAIChatMessage(ctx context.Context, msg *domain.AIChatMessage) error {
	if m.SaveAIChatMessageFunc != nil {
		return m.SaveAIChatMessageFunc(ctx, msg)
	}
	return nil
}
func (m *mockMonRepo) GetAIChatHistory(ctx context.Context, sessionID domain.UUID, limit int) ([]*domain.AIChatMessage, error) {
	if m.GetAIChatHistoryFunc != nil {
		return m.GetAIChatHistoryFunc(ctx, sessionID, limit)
	}
	return nil, nil
}
func (m *mockMonRepo) GetHostChatHistory(ctx context.Context, hostID domain.UUID, limit int) ([]string, error) {
	if m.GetHostChatHistoryFunc != nil {
		return m.GetHostChatHistoryFunc(ctx, hostID, limit)
	}
	return nil, nil
}

// Mock WalletRepository
type mockWalletRepo struct {
	GetByUserIDFunc     func(ctx context.Context, userID domain.UUID) (*domain.Wallet, error)
	CreateFunc          func(ctx context.Context, wallet *domain.Wallet) error
	CreditBalanceFunc   func(ctx context.Context, userID domain.UUID, amount int64) error
	DebitBalanceFunc    func(ctx context.Context, userID domain.UUID, amount int64) error
	FreezeBalanceFunc   func(ctx context.Context, userID domain.UUID, amount int64) error
	UnfreezeBalanceFunc func(ctx context.Context, userID domain.UUID, amount int64) error
	RunInTxFunc         func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *mockWalletRepo) GetByUserID(ctx context.Context, userID domain.UUID) (*domain.Wallet, error) {
	if m.GetByUserIDFunc != nil {
		return m.GetByUserIDFunc(ctx, userID)
	}
	return &domain.Wallet{Balance: 10000}, nil
}
func (m *mockWalletRepo) Create(ctx context.Context, wallet *domain.Wallet) error { return nil }
func (m *mockWalletRepo) CreditBalance(ctx context.Context, userID domain.UUID, amount int64) error {
	if m.CreditBalanceFunc != nil {
		return m.CreditBalanceFunc(ctx, userID, amount)
	}
	return nil
}
func (m *mockWalletRepo) DebitBalance(ctx context.Context, userID domain.UUID, amount int64) error {
	if m.DebitBalanceFunc != nil {
		return m.DebitBalanceFunc(ctx, userID, amount)
	}
	return nil
}
func (m *mockWalletRepo) FreezeBalance(ctx context.Context, userID domain.UUID, amount int64) error {
	return nil
}
func (m *mockWalletRepo) UnfreezeBalance(ctx context.Context, userID domain.UUID, amount int64) error {
	return nil
}
func (m *mockWalletRepo) RunInTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if m.RunInTxFunc != nil {
		return m.RunInTxFunc(ctx, fn)
	}
	// Run standard transaction closure in-place synchronously
	return fn(ctx)
}

// Mock TransactionRepository
type mockTxRepo struct {
	CreateFunc           func(ctx context.Context, tx *domain.Transaction) error
	GetByIDFunc          func(ctx context.Context, id domain.UUID) (*domain.Transaction, error)
	GetByReferenceIDFunc func(ctx context.Context, refID string) (*domain.Transaction, error)
	ListByUserFunc       func(ctx context.Context, userID domain.UUID, txType string, limit, offset int) ([]*domain.Transaction, error)
	UpdateStatusFunc     func(ctx context.Context, id domain.UUID, status string) error
}

func (m *mockTxRepo) Create(ctx context.Context, tx *domain.Transaction) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, tx)
	}
	return nil
}
func (m *mockTxRepo) GetByID(ctx context.Context, id domain.UUID) (*domain.Transaction, error) {
	return nil, nil
}
func (m *mockTxRepo) GetByReferenceID(ctx context.Context, refID string) (*domain.Transaction, error) {
	return nil, nil
}
func (m *mockTxRepo) ListByUser(ctx context.Context, userID domain.UUID, txType string, limit, offset int) ([]*domain.Transaction, error) {
	return nil, nil
}
func (m *mockTxRepo) UpdateStatus(ctx context.Context, id domain.UUID, status string) error {
	return nil
}

// Mock StreamRepositoryForMonetization
type mockStreamRepoForMonetization struct {
	CreateFunc        func(ctx context.Context, stream *domain.Stream) error
	UpdateFunc        func(ctx context.Context, stream *domain.Stream) error
	GetByIDFunc       func(ctx context.Context, id domain.UUID) (*domain.Stream, error)
	GetByRoomIDFunc   func(ctx context.Context, roomID domain.UUID) (*domain.Stream, error)
	GetLiveByHostFunc func(ctx context.Context, hostID domain.UUID) (*domain.Stream, error)
	ListLiveFunc      func(ctx context.Context, limit, offset int) ([]*domain.Stream, error)
}

func (m *mockStreamRepoForMonetization) Create(ctx context.Context, stream *domain.Stream) error {
	return nil
}
func (m *mockStreamRepoForMonetization) Update(ctx context.Context, stream *domain.Stream) error {
	return nil
}
func (m *mockStreamRepoForMonetization) GetByID(ctx context.Context, id domain.UUID) (*domain.Stream, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return &domain.Stream{ID: id, HostID: domain.NewUUID()}, nil
}
func (m *mockStreamRepoForMonetization) GetByRoomID(ctx context.Context, roomID domain.UUID) (*domain.Stream, error) {
	return nil, nil
}
func (m *mockStreamRepoForMonetization) GetLiveByHost(ctx context.Context, hostID domain.UUID) (*domain.Stream, error) {
	return nil, nil
}
func (m *mockStreamRepoForMonetization) ListLive(ctx context.Context, limit, offset int) ([]*domain.Stream, error) {
	if m.ListLiveFunc != nil {
		return m.ListLiveFunc(ctx, limit, offset)
	}
	return []*domain.Stream{}, nil
}

func TestMonetizationUseCase_CreatePaidRoom(t *testing.T) {
	logger := zap.NewNop()
	monRepo := &mockMonRepo{}
	walletRepo := &mockWalletRepo{}
	txRepo := &mockTxRepo{}
	streamRepo := &mockStreamRepoForMonetization{}

	uc := NewMonetizationUseCase(monRepo, walletRepo, txRepo, streamRepo, logger)

	hostID := domain.NewUUID()
	room, err := uc.CreatePaidRoom(context.Background(), hostID, "Premium VIP Room", 5000)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if room.Name != "Premium VIP Room" || room.EntryFeeIDR != 5000 || room.HostID != hostID {
		t.Errorf("Unexpected room details: %+v", room)
	}

	// Validation fails
	_, err = uc.CreatePaidRoom(context.Background(), hostID, "", 5000)
	if err == nil {
		t.Error("Expected error for empty room name, got nil")
	}
}

func TestMonetizationUseCase_JoinPaidRoom(t *testing.T) {
	logger := zap.NewNop()
	hostID := domain.NewUUID()
	userID := domain.NewUUID()
	roomID := domain.NewUUID()

	monRepo := &mockMonRepo{
		GetPaidRoomByIDFunc: func(ctx context.Context, id domain.UUID) (*domain.PaidRoom, error) {
			return &domain.PaidRoom{
				ID:          roomID,
				HostID:      hostID,
				Name:        "Exclusive Private",
				EntryFeeIDR: 2000,
			}, nil
		},
	}

	debitCalled := false
	creditCalled := false
	walletRepo := &mockWalletRepo{
		GetByUserIDFunc: func(ctx context.Context, uID domain.UUID) (*domain.Wallet, error) {
			if uID == userID {
				return &domain.Wallet{UserID: userID, Balance: 5000}, nil
			}
			return &domain.Wallet{UserID: hostID, Balance: 1000}, nil
		},
		DebitBalanceFunc: func(ctx context.Context, uID domain.UUID, amount int64) error {
			if uID == userID && amount == 2000 {
				debitCalled = true
			}
			return nil
		},
		CreditBalanceFunc: func(ctx context.Context, uID domain.UUID, amount int64) error {
			if uID == hostID && amount == 2000 {
				creditCalled = true
			}
			return nil
		},
	}

	txRepo := &mockTxRepo{}
	streamRepo := &mockStreamRepoForMonetization{}

	uc := NewMonetizationUseCase(monRepo, walletRepo, txRepo, streamRepo, logger)

	room, err := uc.JoinPaidRoom(context.Background(), userID, roomID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if room == nil || room.ID != roomID {
		t.Error("Room ID mismatch")
	}

	if !debitCalled || !creditCalled {
		t.Errorf("Expected debit & credit to be called, got debit=%v, credit=%v", debitCalled, creditCalled)
	}
}

func TestMonetizationUseCase_ControlToys(t *testing.T) {
	logger := zap.NewNop()
	userID := domain.NewUUID()
	streamID := domain.NewUUID()
	hostID := domain.NewUUID()

	monRepo := &mockMonRepo{
		GetHostDevicesFunc: func(ctx context.Context, hID domain.UUID) ([]*domain.HostDevice, error) {
			if hID == hostID {
				return []*domain.HostDevice{
					{
						ID:         domain.NewUUID(),
						HostID:     hostID,
						DeviceName: "Lovense Lush 3",
						DeviceID:   "device_lush",
						APIToken:   "api_token_test",
					},
				}, nil
			}
			return []*domain.HostDevice{}, nil
		},
	}

	streamRepo := &mockStreamRepoForMonetization{
		GetByIDFunc: func(ctx context.Context, id domain.UUID) (*domain.Stream, error) {
			return &domain.Stream{ID: streamID, HostID: hostID}, nil
		},
	}

	debitCalled := false
	creditCalled := false
	walletRepo := &mockWalletRepo{
		GetByUserIDFunc: func(ctx context.Context, uID domain.UUID) (*domain.Wallet, error) {
			return &domain.Wallet{UserID: userID, Balance: 5000}, nil
		},
		DebitBalanceFunc: func(ctx context.Context, uID domain.UUID, amount int64) error {
			if uID == userID && amount == 1000 {
				debitCalled = true
			}
			return nil
		},
		CreditBalanceFunc: func(ctx context.Context, uID domain.UUID, amount int64) error {
			if uID == hostID && amount == 1000 {
				creditCalled = true
			}
			return nil
		},
	}

	txRepo := &mockTxRepo{}

	uc := NewMonetizationUseCase(monRepo, walletRepo, txRepo, streamRepo, logger)

	// Control with tips = 1000
	msg, err := uc.ControlToys(context.Background(), userID, streamID, "Vibrate:3", 10, 1000)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !debitCalled || !creditCalled {
		t.Errorf("Expected tipping debit/credit to be called: debit=%v, credit=%v", debitCalled, creditCalled)
	}

	if msg == "" {
		t.Error("Expected descriptive success message, got empty string")
	}
}

func TestMonetizationUseCase_SendAIChatMessage(t *testing.T) {
	logger := zap.NewNop()
	userID := domain.NewUUID()
	hostID := domain.NewUUID()
	sessionID := domain.NewUUID()

	monRepo := &mockMonRepo{
		GetAIChatSessionFunc: func(ctx context.Context, uID, hID domain.UUID) (*domain.AIChatSession, error) {
			return &domain.AIChatSession{ID: sessionID, UserID: userID, HostID: hostID}, nil
		},
		SaveAIChatMessageFunc: func(ctx context.Context, msg *domain.AIChatMessage) error {
			return nil
		},
		GetHostChatHistoryFunc: func(ctx context.Context, hID domain.UUID, limit int) ([]string, error) {
			return []string{"Halo kak!", "Makasih banget ya dukungannya!"}, nil
		},
	}

	debitCalled := false
	creditCalled := false
	walletRepo := &mockWalletRepo{
		GetByUserIDFunc: func(ctx context.Context, uID domain.UUID) (*domain.Wallet, error) {
			return &domain.Wallet{UserID: userID, Balance: 1000}, nil
		},
		DebitBalanceFunc: func(ctx context.Context, uID domain.UUID, amount int64) error {
			if uID == userID && amount == 500 {
				debitCalled = true
			}
			return nil
		},
		CreditBalanceFunc: func(ctx context.Context, uID domain.UUID, amount int64) error {
			if uID == hostID && amount == 500 {
				creditCalled = true
			}
			return nil
		},
	}

	txRepo := &mockTxRepo{}
	streamRepo := &mockStreamRepoForMonetization{} // Live list is empty -> creator offline

	uc := NewMonetizationUseCase(monRepo, walletRepo, txRepo, streamRepo, logger)

	aiMsg, err := uc.SendAIChatMessage(context.Background(), userID, hostID, "Halo cantik, apa kabar?")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if aiMsg == nil || aiMsg.SenderType != "ai" {
		t.Fatalf("Expected AI message response, got %+v", aiMsg)
	}

	if !debitCalled || !creditCalled {
		t.Error("Expected AI chat passive earning fee flow to execute")
	}

	// Content filter test
	_, err = uc.SendAIChatMessage(context.Background(), userID, hostID, "Halo bangsat, apa kabar?")
	if err == nil {
		t.Error("Expected bad word to trigger content filtering error, but it passed")
	}
}
