package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	gorilla_ws "github.com/gorilla/websocket"
	"nvide-live/internal/domain"
	"nvide-live/pkg/broker"
	"nvide-live/pkg/redis"

	"go.uber.org/zap"
)

// RoomSettings represents room-specific moderation options
type RoomSettings struct {
	HostID          string
	SlowModeSeconds int
	ChatMode        string // "slow", "followers_only", "subscribers_only", "level_gate"
	MinLevelToChat  int
	PinnedMessage   *WSMessage
}

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	clients      map[*Client]bool
	rooms        map[string]map[*Client]bool // roomID -> clients
	broadcast    chan *WSMessage
	register     chan *Client
	unregister   chan *Client
	roomSettings map[string]*RoomSettings // roomID -> settings

	db           *pgxpool.Pool
	broker       broker.Broker
	redisClient  *redis.Client
	logger       *zap.Logger
	moderationUC domain.ModerationUseCase
	mu           sync.RWMutex
}

// WSMessage represents the standard JSON message format
type WSMessage struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp string      `json:"timestamp"`
	RoomID    string      `json:"-"` // Not serialized, used for routing
	UserID    string      `json:"-"` // Not serialized, used for rate limiting
}

// NewHub creates a new Hub instance
func NewHub(db *pgxpool.Pool, b broker.Broker, r *redis.Client, logger *zap.Logger) *Hub {
	return &Hub{
		clients:      make(map[*Client]bool),
		rooms:        make(map[string]map[*Client]bool),
		broadcast:    make(chan *WSMessage),
		register:     make(chan *Client, 1024),
		unregister:   make(chan *Client, 1024),
		roomSettings: make(map[string]*RoomSettings),
		db:           db,
		broker:       b,
		redisClient:  r,
		logger:       logger,
	}
}

func (h *Hub) SetModerationUseCase(uc domain.ModerationUseCase) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.moderationUC = uc
}

// Run starts the hub loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			if _, ok := h.rooms[client.roomID]; !ok {
				h.rooms[client.roomID] = make(map[*Client]bool)
				// Subscribe to broker for this room if it's the first client
				h.subscribeToRoom(client.roomID)
			}
			h.rooms[client.roomID][client] = true
			h.mu.Unlock()

			// Emit join message
			h.emitSystemMessage(client.roomID, fmt.Sprintf("User %s joined the room", client.userID))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				if _, ok := h.rooms[client.roomID]; ok {
					delete(h.rooms[client.roomID], client)
					if len(h.rooms[client.roomID]) == 0 {
						delete(h.rooms, client.roomID)
						delete(h.roomSettings, client.roomID) // Prevent room settings memory leak!
						
						// Periodic force GC asynchronously after stream/room end
						go func(rid string) {
							time.Sleep(3 * time.Second)
							h.logger.Info("Forcing garbage collection to free memory after room end", zap.String("room_id", rid))
							runtime.GC()
						}(client.roomID)
					}
				}
				// Emit leave message
				go h.emitSystemMessage(client.roomID, fmt.Sprintf("User %s left the room", client.userID))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			// 1. Fetch Room Settings (HostID, SlowMode, ChatMode, LevelGate, etc.)
			settings := h.getRoomSettings(message.RoomID)

			// 2. Check if the user is Kicked (Banned 30m) from this room
			kickKey := fmt.Sprintf("room:kick:%s:%s", message.RoomID, message.UserID)
			isKicked, _ := h.redisClient.GetClient().Exists(context.Background(), kickKey).Result()
			if isKicked > 0 {
				h.sendPrivateError(message.UserID, message.RoomID, "KICKED", "Anda telah di-kick dari ruangan ini.")
				continue
			}

			// 3. Handle Moderation Commands (Mute, Kick, Pin, ChatSettings)
			if message.Type == "mute" || message.Type == "kick" || message.Type == "pin" || message.Type == "chat_settings" {
				// Check if sender is Host or Moderator
				isAuthorized := message.UserID == settings.HostID
				if !isAuthorized {
					modKey := fmt.Sprintf("room:moderator:%s:%s", message.RoomID, message.UserID)
					modExists, _ := h.redisClient.GetClient().Exists(context.Background(), modKey).Result()
					if modExists > 0 {
						isAuthorized = true
					} else {
						var exists bool
						err := h.db.QueryRow(context.Background(), 
							"SELECT EXISTS(SELECT 1 FROM stream_moderators WHERE stream_id = (SELECT id FROM streams WHERE room_id = $1 OR id = $1 LIMIT 1) AND user_id = $2)", 
							message.RoomID, message.UserID).Scan(&exists)
						if err == nil && exists {
							isAuthorized = true
							h.redisClient.GetClient().Set(context.Background(), modKey, "1", 24*time.Hour)
						}
					}
				}

				if !isAuthorized {
					h.sendPrivateError(message.UserID, message.RoomID, "UNAUTHORIZED", "Anda tidak memiliki izin moderator.")
					continue
				}

				h.executeModeratorCommand(message, settings)
				continue
			}

			// 4. Check if the user is Muted in this room
			muteKey := fmt.Sprintf("room:mute:%s:%s", message.RoomID, message.UserID)
			isMuted, _ := h.redisClient.GetClient().Exists(context.Background(), muteKey).Result()
			if isMuted > 0 {
				h.sendPrivateError(message.UserID, message.RoomID, "MUTED", "Anda sedang di-mute di ruangan ini.")
				continue
			}

			// 5. Rate limit check (Token Bucket via Redis Lua script)
			limited, retryAfter := h.checkTokenBucketRateLimit(message.UserID, message.RoomID)
			if limited {
				h.logger.Warn("User rate limited", zap.String("user_id", message.UserID), zap.String("room_id", message.RoomID))
				h.sendPrivateErrorWithRetry(message.UserID, message.RoomID, "429", "Rate limit exceeded. Please wait.", retryAfter)
				continue
			}

			// 6. Check if user is sending a normal chat message and enforce slow-mode / VIP privilege
			if message.Type == "chat" {
				// Sync chat auto-moderation pipeline
				chatText, ok := message.Payload.(string)
				if ok && h.moderationUC != nil {
					decision, err := h.moderationUC.EvaluateChatMessage(context.Background(), domain.UUID(message.UserID), domain.UUID(message.RoomID), chatText)
					if err == nil && decision.ActionTaken != "pass" {
						if decision.Blocked || decision.Muted || decision.Kicked || decision.Banned {
							// Block message from broadcast
							continue
						}
					}
				}

				userLevel, chatColor := h.getUserLevelAndColor(message.UserID)
				
				// Enforce Chat Level Gate
				if settings.ChatMode == "level_gate" && userLevel < settings.MinLevelToChat {
					h.sendPrivateError(message.UserID, message.RoomID, "LEVEL_TOO_LOW", fmt.Sprintf("Level minimal untuk chat di room ini adalah level %d.", settings.MinLevelToChat))
					continue
				}

				// VIP Slow Mode Bypass Check: Top 3 Gifters bypass slow-mode!
				isVIP := false
				var rank int64 = -1
				
				contributorsKey := fmt.Sprintf("room:contributors:%s", message.RoomID)
				rankVal, err := h.redisClient.GetClient().ZRevRank(context.Background(), contributorsKey, message.UserID).Result()
				if err == nil {
					rank = rankVal
					if rank < 3 {
						isVIP = true
					}
				}
				if userLevel >= 10 {
					isVIP = true
				}

				// Enforce Slow Mode
				if settings.SlowModeSeconds > 0 && !isVIP && message.UserID != settings.HostID {
					lastMsgKey := fmt.Sprintf("room:last_msg:%s:%s", message.RoomID, message.UserID)
					lastMsgStr, err := h.redisClient.GetClient().Get(context.Background(), lastMsgKey).Result()
					if err == nil && lastMsgStr != "" {
						lastTime, _ := strconv.ParseInt(lastMsgStr, 10, 64)
						diff := time.Now().Unix() - lastTime
						if diff < int64(settings.SlowModeSeconds) {
							h.sendPrivateError(message.UserID, message.RoomID, "SLOW_MODE", fmt.Sprintf("Mohon tunggu %d detik sebelum mengirim pesan lagi.", int64(settings.SlowModeSeconds)-diff))
							continue
						}
					}
					h.redisClient.GetClient().Set(context.Background(), lastMsgKey, strconv.FormatInt(time.Now().Unix(), 10), time.Duration(settings.SlowModeSeconds)*time.Second)
				}

				// Enrich payload with Level, ChatColor, VIP status, and DisplayName!
				var displayName string
				_ = h.db.QueryRow(context.Background(), "SELECT username FROM users WHERE id = $1", message.UserID).Scan(&displayName)

				// Simpan pesan ke tabel comments (migrasi 003_social_features.sql)
				if ok {
					uID, errUser := domain.FromString(message.UserID)
					rID, errRoom := domain.FromString(message.RoomID)
					if errUser == nil && errRoom == nil {
						commentID := domain.NewUUID()
						_, errDb := h.db.Exec(context.Background(),
							"INSERT INTO comments (id, user_id, content_id, content_type, content, created_at, updated_at) VALUES ($1, $2, $3, 'stream', $4, NOW(), NOW())",
							commentID, uID, rID, chatText)
						if errDb != nil {
							h.logger.Error("Failed to save comment to database", zap.Error(errDb))
						} else {
							h.logger.Debug("Comment saved to database", zap.String("comment_id", commentID.String()))
						}
					}
				}

				enrichedPayload := map[string]interface{}{
					"user_id":      message.UserID,
					"username":     displayName,
					"content":      message.Payload,
					"user_level":   userLevel,
					"chat_color":   chatColor,
					"is_vip":       isVIP,
					"vip_rank":     rank + 1,
				}
				message.Payload = enrichedPayload
			}

			// Broadcast to broker
			topic := "room:" + message.RoomID
			msgBytes, err := json.Marshal(message)
			if err != nil {
				h.logger.Error("Failed to marshal message", zap.Error(err))
				continue
			}
			h.broker.Publish(context.Background(), topic, string(msgBytes))
		}
	}
}

// subscribeToRoom subscribes the hub to a room's topic on the broker
func (h *Hub) subscribeToRoom(roomID string) {
	topic := "room:" + roomID
	err := h.broker.Subscribe(context.Background(), topic, func(message string) {
		h.mu.RLock()
		defer h.mu.RUnlock()

		if clients, ok := h.rooms[roomID]; ok {
			msgBytes := []byte(message)
			for client := range clients {
				select {
				case client.send <- msgBytes:
				default:
					close(client.send)
					delete(h.clients, client)
					delete(h.rooms[roomID], client)
				}
			}
		}
	})
	if err != nil {
		h.logger.Error("Failed to subscribe to topic", zap.String("topic", topic), zap.Error(err))
	}
}

// emitSystemMessage sends a system message to a room via broker
func (h *Hub) emitSystemMessage(roomID string, content string) {
	msg := &WSMessage{
		Type: "system",
		Payload: map[string]string{
			"content": content,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}
	msgBytes, _ := json.Marshal(msg)
	h.broker.Publish(context.Background(), "room:"+roomID, string(msgBytes))
}

// checkTokenBucketRateLimit checks if a user has exceeded 3 msgs/second in a room
// Using an atomic Redis Lua Script Token Bucket algorithm
func (h *Hub) checkTokenBucketRateLimit(userID, roomID string) (bool, float64) {
	if h.redisClient == nil || userID == "" {
		return false, 0
	}
	ctx := context.Background()
	key := fmt.Sprintf("ratelimit:ws:%s:%s", roomID, userID)
	
	// Token Bucket LUA script
	// ARGV[1] = limit/capacity (3), ARGV[2] = current unix timestamp
	luaScript := `
		local key = KEYS[1]
		local limit = tonumber(ARGV[1])
		local current_time = tonumber(ARGV[2])
		local fill_rate = limit
		local capacity = limit

		local data = redis.call('HMGET', key, 'tokens', 'last_updated')
		local tokens = tonumber(data[1])
		local last_updated = tonumber(data[2])

		if not tokens then
			tokens = capacity
			last_updated = current_time
		else
			local elapsed = math.max(0, current_time - last_updated)
			tokens = math.min(capacity, tokens + elapsed * fill_rate)
		end

		if tokens < 1 then
			local retry_after = (1 - tokens) / fill_rate
			return {0, tostring(retry_after)}
		else
			tokens = tokens - 1
			redis.call('HMSET', key, 'tokens', tokens, 'last_updated', current_time)
			redis.call('EXPIRE', key, 2)
			return {1, "0"}
		end
	`

	nowSec := float64(time.Now().UnixNano()) / 1e9
	res, err := h.redisClient.GetClient().Eval(ctx, luaScript, []string{key}, 3, nowSec).Result()
	if err != nil {
		h.logger.Error("Token Bucket Lua eval error", zap.Error(err))
		return false, 0 // Fail open
	}

	slice, ok := res.([]interface{})
	if !ok || len(slice) < 2 {
		return false, 0
	}

	allowed := slice[0].(int64)
	retryAfterStr := slice[1].(string)
	retryAfter, _ := strconv.ParseFloat(retryAfterStr, 64)

	return allowed == 0, retryAfter
}

// BroadcastToRoom sends raw bytes to all clients in a specific room via the broker
func (h *Hub) BroadcastToRoom(roomID string, data []byte) {
	topic := "room:" + roomID
	h.broker.Publish(context.Background(), topic, string(data))
}

func (h *Hub) getRoomSettings(roomID string) *RoomSettings {
	h.mu.RLock()
	settings, exists := h.roomSettings[roomID]
	h.mu.RUnlock()
	if exists {
		return settings
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	// Double check
	settings, exists = h.roomSettings[roomID]
	if exists {
		return settings
	}

	var hostID string
	var slowMode int
	var chatMode string
	var minLevel int
	query := `
		SELECT host_id::text, chat_slow_mode_seconds, chat_mode, min_level_to_enter 
		FROM streams 
		WHERE room_id = $1 OR id = $1
		LIMIT 1
	`
	err := h.db.QueryRow(context.Background(), query, roomID).Scan(&hostID, &slowMode, &chatMode, &minLevel)
	if err != nil {
		settings = &RoomSettings{
			HostID:          "",
			SlowModeSeconds: 0,
			ChatMode:        "public",
			MinLevelToChat:  1,
		}
	} else {
		settings = &RoomSettings{
			HostID:          hostID,
			SlowModeSeconds: slowMode,
			ChatMode:        chatMode,
			MinLevelToChat:  minLevel,
		}
	}

	if h.roomSettings == nil {
		h.roomSettings = make(map[string]*RoomSettings)
	}
	h.roomSettings[roomID] = settings
	return settings
}

func (h *Hub) getUserLevelAndColor(userID string) (int, string) {
	ctx := context.Background()
	levelKey := fmt.Sprintf("user:level:%s", userID)
	colorKey := fmt.Sprintf("user:color:%s", userID)

	levelStr, err1 := h.redisClient.GetClient().Get(ctx, levelKey).Result()
	colorStr, err2 := h.redisClient.GetClient().Get(ctx, colorKey).Result()

	if err1 == nil && err2 == nil && levelStr != "" {
		level, _ := strconv.Atoi(levelStr)
		return level, colorStr
	}

	var userLevel int
	var role string
	err := h.db.QueryRow(ctx, "SELECT user_level, role FROM users WHERE id = $1", userID).Scan(&userLevel, &role)
	if err != nil {
		return 1, "#FFFFFF"
	}

	chatColor := "#FFFFFF"
	if role == "host" {
		chatColor = "#FF4081"
	} else if userLevel >= 50 {
		chatColor = "#FFD700"
	} else if userLevel >= 25 {
		chatColor = "#A020F0"
	} else if userLevel >= 10 {
		chatColor = "#00E5FF"
	}

	h.redisClient.GetClient().Set(ctx, levelKey, strconv.Itoa(userLevel), 10*time.Minute)
	h.redisClient.GetClient().Set(ctx, colorKey, chatColor, 10*time.Minute)

	return userLevel, chatColor
}

func (h *Hub) sendPrivateError(userID, roomID, code, message string) {
	h.sendPrivateErrorWithRetry(userID, roomID, code, message, 0)
}

func (h *Hub) sendPrivateErrorWithRetry(userID, roomID, code, message string, retryAfter float64) {
	payloadMap := map[string]interface{}{
		"code":    code,
		"message": message,
	}
	if retryAfter > 0 {
		payloadMap["retry_after"] = retryAfter
	}

	errPayload := map[string]interface{}{
		"type":      "error",
		"payload":   payloadMap,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	msgBytes, _ := json.Marshal(errPayload)

	h.mu.RLock()
	defer h.mu.RUnlock()
	if clients, ok := h.rooms[roomID]; ok {
		for client := range clients {
			if client.userID == userID {
				select {
				case client.send <- msgBytes:
				default:
				}
			}
		}
	}
}

func (h *Hub) broadcastToRoomRaw(roomID string, message *WSMessage) {
	topic := "room:" + roomID
	msgBytes, _ := json.Marshal(message)
	h.broker.Publish(context.Background(), topic, string(msgBytes))
}

func (h *Hub) executeModeratorCommand(message *WSMessage, settings *RoomSettings) {
	ctx := context.Background()
	payloadMap, ok := message.Payload.(map[string]interface{})
	if !ok {
		return
	}

	targetUserID, _ := payloadMap["target_user_id"].(string)

	switch message.Type {
	case "mute":
		durationSec := 1800
		if d, ok := payloadMap["duration_seconds"].(float64); ok {
			durationSec = int(d)
		}
		muteKey := fmt.Sprintf("room:mute:%s:%s", message.RoomID, targetUserID)
		h.redisClient.GetClient().Set(ctx, muteKey, "1", time.Duration(durationSec)*time.Second)

		broadcastMsg := &WSMessage{
			Type: "user_muted",
			Payload: map[string]interface{}{
				"target_user_id":   targetUserID,
				"duration_seconds": durationSec,
				"message":          fmt.Sprintf("User %s telah di-mute selama %d menit.", targetUserID, durationSec/60),
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
		h.broadcastToRoomRaw(message.RoomID, broadcastMsg)

	case "kick":
		kickKey := fmt.Sprintf("room:kick:%s:%s", message.RoomID, targetUserID)
		h.redisClient.GetClient().Set(ctx, kickKey, "1", 30*time.Minute)

		broadcastMsg := &WSMessage{
			Type: "user_kicked",
			Payload: map[string]interface{}{
				"target_user_id": targetUserID,
				"message":        fmt.Sprintf("User %s telah di-kick dari ruangan ini.", targetUserID),
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
		h.broadcastToRoomRaw(message.RoomID, broadcastMsg)

		h.mu.Lock()
		if clients, ok := h.rooms[message.RoomID]; ok {
			for client := range clients {
				if client.userID == targetUserID {
					client.conn.Close()
					delete(h.clients, client)
					delete(h.rooms[message.RoomID], client)
				}
			}
		}
		h.mu.Unlock()

	case "pin":
		pinnedContent, _ := payloadMap["content"].(string)
		pinKey := fmt.Sprintf("room:pinned:%s", message.RoomID)
		h.redisClient.GetClient().Set(ctx, pinKey, pinnedContent, 24*time.Hour)

		settings.PinnedMessage = message

		broadcastMsg := &WSMessage{
			Type: "message_pinned",
			Payload: map[string]interface{}{
				"content": pinnedContent,
				"message": "Pesan disematkan oleh moderator.",
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
		h.broadcastToRoomRaw(message.RoomID, broadcastMsg)

	case "chat_settings":
		if mode, ok := payloadMap["chat_mode"].(string); ok {
			settings.ChatMode = mode
		}
		if slowSec, ok := payloadMap["slow_mode_seconds"].(float64); ok {
			settings.SlowModeSeconds = int(slowSec)
		}
		if minLvl, ok := payloadMap["min_level_to_chat"].(float64); ok {
			settings.MinLevelToChat = int(minLvl)
		}

		broadcastMsg := &WSMessage{
			Type: "chat_settings_updated",
			Payload: map[string]interface{}{
				"chat_mode":         settings.ChatMode,
				"slow_mode_seconds": settings.SlowModeSeconds,
				"min_level_to_chat": settings.MinLevelToChat,
			},
			Timestamp: time.Now().Format(time.RFC3339),
		}
		h.broadcastToRoomRaw(message.RoomID, broadcastMsg)
	}
}

// Stop shuts down the hub and disconnects all clients gracefully (Fase 5)
func (h *Hub) Stop() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.logger.Info("Shutting down WebSocket Hub, draining active clients...", zap.Int("active_clients", len(h.clients)))
	for client := range h.clients {
		// Send a clean WS close control message
		_ = client.conn.WriteControl(
			gorilla_ws.CloseMessage,
			gorilla_ws.FormatCloseMessage(gorilla_ws.CloseNormalClosure, "Server shutting down gracefully"),
			time.Now().Add(1*time.Second),
		)
		client.conn.Close()
		delete(h.clients, client)
	}

	// Reset rooms and settings maps
	h.rooms = make(map[string]map[*Client]bool)
	h.roomSettings = make(map[string]*RoomSettings)
	h.logger.Info("WebSocket Hub drained and stopped successfully")
}

// IsStreamRoom checks if a room ID belongs to a live stream
func (h *Hub) IsStreamRoom(roomID string) bool {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM streams WHERE id = $1 OR room_id = $1 OR room_id::text = $1)"
	err := h.db.QueryRow(context.Background(), query, roomID).Scan(&exists)
	return err == nil && exists
}

// GetBroker returns the underlying broker instance
func (h *Hub) GetBroker() broker.Broker {
	return h.broker
}
