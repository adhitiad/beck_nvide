//go:build wireinject
// +build wireinject

package server

import (
	"github.com/google/wire"
	"go.uber.org/zap"
	"nvide-live/pkg/config"
	"nvide-live/pkg/database"
	"nvide-live/pkg/redis"
	"nvide-live/internal/domain"
	"nvide-live/internal/usecase"
	"nvide-live/internal/webrtc"
	"nvide-live/internal/websocket"
	workerV1 "nvide-live/internal/worker"
	"nvide-live/pkg/worker"
	"nvide-live/pkg/rbac"
	"github.com/gorilla/mux"
)

// AppContainer holds the initialized core dependencies for the server.
type AppContainer struct {
	Config             *config.Config
	DB                 *database.DB
	Redis              *redis.Client
	Logger             *zap.Logger
	Router             *mux.Router
	WaitRoomHub        *websocket.WaitRoomHub
	WSHub              *websocket.Hub
	ModerationUseCase  domain.ModerationUseCase
	ScheduleUseCase    domain.LiveScheduleUseCase
	PrivateChatUseCase domain.PrivateChatUsecase
	StreamUseCase      *usecase.StreamUseCase
	VIPUseCase         *usecase.VIPUseCase
	HostLevelUseCase   *usecase.HostLevelUseCase
	CryptoUseCase      *usecase.CryptoUseCase
	CryptoRepo         domain.CryptoRepository
	CryptoMonitor      *workerV1.CryptoMonitor
	WorkerPool         *worker.WorkerPool
	TrendingUseCase    *usecase.TrendingUseCase
	VODUseCase         *usecase.VODUseCase
	LevelingUseCase    *usecase.LevelingUseCase
	WebRTCRoomManager  *webrtc.RoomManager
	RBACManager        *rbac.Manager
	RoleRepo           domain.RoleRepository
}

// InitializeApp configures the main dependency graph.
func InitializeApp(logger *zap.Logger) (*AppContainer, error) {
	wire.Build(
		AppSet,
		wire.Struct(new(AppContainer), "*"),
	)
	return &AppContainer{}, nil
}
