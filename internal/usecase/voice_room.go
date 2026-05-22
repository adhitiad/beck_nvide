package usecase

import (
	"context"

	"go.uber.org/zap"

	"nvide-live/internal/domain"
)

type VoiceRoomUseCase struct {
	roomRepo domain.VoiceRoomRepository
	logger   *zap.Logger
}

func NewVoiceRoomUseCase(roomRepo domain.VoiceRoomRepository, logger *zap.Logger) *VoiceRoomUseCase {
	return &VoiceRoomUseCase{roomRepo: roomRepo, logger: logger}
}

func (uc *VoiceRoomUseCase) CreateRoom(ctx context.Context, hostID domain.UUID, title, description string, maxSpeakers int) (*domain.VoiceRoom, error) {
	if maxSpeakers <= 0 || maxSpeakers > 16 {
		maxSpeakers = 8
	}
	room := &domain.VoiceRoom{
		ID:          domain.NewUUID(),
		HostID:      hostID,
		Title:       title,
		Description: description,
		MaxSpeakers: maxSpeakers,
		Status:      domain.VoiceRoomStatusActive,
	}
	if err := uc.roomRepo.Create(ctx, room); err != nil {
		return nil, err
	}
	// Auto-add host as participant
	p := &domain.VoiceRoomParticipant{
		ID:     domain.NewUUID(),
		RoomID: room.ID,
		UserID: hostID,
		Role:   domain.VoiceRoleHost,
	}
	_ = uc.roomRepo.AddParticipant(ctx, p)
	return room, nil
}

func (uc *VoiceRoomUseCase) JoinRoom(ctx context.Context, userID, roomID domain.UUID) error {
	room, err := uc.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeNotFound, "room not found", err)
	}
	if room.Status != domain.VoiceRoomStatusActive {
		return domain.NewDomainError(domain.ErrCodeConflict, "room has ended", nil)
	}
	p := &domain.VoiceRoomParticipant{
		ID:     domain.NewUUID(),
		RoomID: roomID,
		UserID: userID,
		Role:   domain.VoiceRoleListener,
	}
	return uc.roomRepo.AddParticipant(ctx, p)
}

func (uc *VoiceRoomUseCase) LeaveRoom(ctx context.Context, userID, roomID domain.UUID) error {
	return uc.roomRepo.RemoveParticipant(ctx, roomID, userID)
}

func (uc *VoiceRoomUseCase) RequestStage(ctx context.Context, userID, roomID domain.UUID) error {
	// For now, directly promote to speaker (host approval can be added via WS)
	room, err := uc.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeNotFound, "room not found", err)
	}
	speakerCount, _ := uc.roomRepo.CountSpeakers(ctx, roomID)
	if speakerCount >= room.MaxSpeakers {
		return domain.NewDomainError(domain.ErrCodeConflict, "speaker slots are full", nil)
	}
	return uc.roomRepo.UpdateParticipantRole(ctx, roomID, userID, domain.VoiceRoleSpeaker)
}

func (uc *VoiceRoomUseCase) ApproveStage(ctx context.Context, hostID, userID, roomID domain.UUID) error {
	room, err := uc.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeNotFound, "room not found", err)
	}
	if room.HostID != hostID {
		return domain.NewDomainError(domain.ErrCodeForbidden, "only host can approve stage requests", nil)
	}
	return uc.roomRepo.UpdateParticipantRole(ctx, roomID, userID, domain.VoiceRoleSpeaker)
}

func (uc *VoiceRoomUseCase) RemoveFromStage(ctx context.Context, hostID, userID, roomID domain.UUID) error {
	room, err := uc.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeNotFound, "room not found", err)
	}
	if room.HostID != hostID {
		return domain.NewDomainError(domain.ErrCodeForbidden, "only host can remove from stage", nil)
	}
	return uc.roomRepo.UpdateParticipantRole(ctx, roomID, userID, domain.VoiceRoleListener)
}

func (uc *VoiceRoomUseCase) EndRoom(ctx context.Context, hostID, roomID domain.UUID) error {
	room, err := uc.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeNotFound, "room not found", err)
	}
	if room.HostID != hostID {
		return domain.NewDomainError(domain.ErrCodeForbidden, "only host can end room", nil)
	}
	return uc.roomRepo.EndRoom(ctx, roomID)
}

func (uc *VoiceRoomUseCase) ListActive(ctx context.Context, limit, offset int) ([]*domain.VoiceRoom, error) {
	return uc.roomRepo.ListActive(ctx, limit, offset)
}

func (uc *VoiceRoomUseCase) GetRoom(ctx context.Context, roomID domain.UUID) (*domain.VoiceRoom, error) {
	room, err := uc.roomRepo.GetByID(ctx, roomID)
	if err != nil {
		return nil, err
	}
	participants, _ := uc.roomRepo.ListParticipants(ctx, roomID)
	room.Participants = participants
	return room, nil
}
