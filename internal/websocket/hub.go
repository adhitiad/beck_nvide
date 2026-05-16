package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"nvide-live/pkg/broker"
	"nvide-live/pkg/redis"

	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Hub maintains the set of active clients and broadcasts messages
type Hub struct {
	clients    map[*Client]bool
	rooms      map[string]map[*Client]bool // roomID -> clients
	broadcast  chan *WSMessage
	register   chan *Client
	unregister chan *Client

	broker      broker.Broker
	redisClient *redis.Client
	logger      *zap.Logger
	mu          sync.RWMutex
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
func NewHub(b broker.Broker, r *redis.Client, logger *zap.Logger) *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		rooms:       make(map[string]map[*Client]bool),
		broadcast:   make(chan *WSMessage),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broker:      b,
		redisClient: r,
		logger:      logger,
	}
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
					}
				}
				// Emit leave message
				go h.emitSystemMessage(client.roomID, fmt.Sprintf("User %s left the room", client.userID))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			// Rate limit check
			if h.isRateLimited(message.UserID, message.RoomID) {
				h.logger.Warn("User rate limited", zap.String("user_id", message.UserID), zap.String("room_id", message.RoomID))
				continue
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

// isRateLimited checks if a user has exceeded 3 msgs/second in a room
// Using Redis Sliding Window rate limiting
func (h *Hub) isRateLimited(userID, roomID string) bool {
	if h.redisClient == nil || userID == "" {
		return false
	}
	ctx := context.Background()
	key := fmt.Sprintf("ratelimit:ws:%s:%s", roomID, userID)
	now := time.Now().UnixNano()
	windowStart := now - time.Second.Nanoseconds()

	// ZREMRANGEBYSCORE key -inf windowStart
	h.redisClient.GetClient().ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%d", windowStart))

	// ZCOUNT key -inf +inf
	count, err := h.redisClient.GetClient().ZCard(ctx, key).Result()
	if err != nil {
		h.logger.Error("Redis ZCard error", zap.Error(err))
		return false // Fail open
	}

	if count >= 3 {
		return true // Rate limited
	}

	// ZADD key now now
	h.redisClient.GetClient().ZAdd(ctx, key, goredis.Z{Score: float64(now), Member: now})
	// Expire to cleanup memory automatically
	h.redisClient.GetClient().Expire(ctx, key, 2*time.Second)

	return false
}

// BroadcastToRoom sends raw bytes to all clients in a specific room via the broker
func (h *Hub) BroadcastToRoom(roomID string, data []byte) {
	topic := "room:" + roomID
	h.broker.Publish(context.Background(), topic, string(data))
}
