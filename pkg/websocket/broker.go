package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
	"nvide-live/pkg/redis"
	"nvide-live/pkg/uuid"
)

// BrokerMessage defines the standard JSON payload transferred through the broker
type BrokerMessage struct {
	Event     string          `json:"event"`
	RoomID    string          `json:"room_id"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp int64           `json:"timestamp"`
	NodeID    string          `json:"node_id"`
}

// BrokerHandler is a function that processes incoming BrokerMessage
type BrokerHandler func(ctx context.Context, msg *BrokerMessage)

// WebsocketBroker coordinates messages across multiple server instances using Redis Pub/Sub
type WebsocketBroker struct {
	nodeID      string
	redisClient *redis.Client
	globalTopic string
	logger      *zap.Logger
	ctx         context.Context
	cancel      context.CancelFunc

	mu          sync.RWMutex
	handlers    map[string][]BrokerHandler // room_id -> list of handlers
	isClosed    bool
	isFallback  bool

	// In-memory channel for local loopback when in fallback mode
	localBus    chan *BrokerMessage
}

// NewWebsocketBroker creates a new WebsocketBroker
func NewWebsocketBroker(redisClient *redis.Client, logger *zap.Logger) *WebsocketBroker {
	ctx, cancel := context.WithCancel(context.Background())
	nodeID := uuid.NewV7()
	
	broker := &WebsocketBroker{
		nodeID:      nodeID,
		redisClient: redisClient,
		globalTopic: "ws_pubsub_global",
		logger:      logger.With(zap.String("node_id", nodeID)),
		ctx:         ctx,
		cancel:      cancel,
		handlers:    make(map[string][]BrokerHandler),
		localBus:    make(chan *BrokerMessage, 1000),
	}

	// Verify Redis health
	if redisClient == nil || redisClient.Health(ctx) != nil {
		broker.isFallback = true
		logger.Warn("Redis is unavailable. Falling back to in-memory Broker mode.")
	}

	return broker
}

// Start launches the pub/sub listener
func (b *WebsocketBroker) Start() {
	if b.isFallback {
		go b.listenInMemory()
		b.logger.Info("In-memory broker loop started")
		return
	}

	go b.listenRedis()
	b.logger.Info("Redis Pub/Sub broker loop started")
}

// Publish distributes a message to all instances (including self, but self will skip echo loop)
func (b *WebsocketBroker) Publish(ctx context.Context, roomID string, event string, payload json.RawMessage) error {
	msg := &BrokerMessage{
		Event:     event,
		RoomID:    roomID,
		Payload:   payload,
		Timestamp: time.Now().UnixMilli(),
		NodeID:    b.nodeID,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Check if Redis is up, otherwise trigger fallback dynamically
	if !b.isFallback && b.redisClient != nil && b.redisClient.Health(ctx) == nil {
		err = b.redisClient.GetClient().Publish(ctx, b.globalTopic, string(msgBytes)).Err()
		if err == nil {
			return nil
		}
		b.logger.Warn("Failed to publish to Redis, dynamically entering fallback mode", zap.Error(err))
	}

	// Fallback to local in-memory
	select {
	case b.localBus <- msg:
		return nil
	default:
		b.logger.Warn("Local message bus is full, message discarded")
		return errors.New("local message bus full")
	}
}

// Subscribe registers a handler for a specific room
func (b *WebsocketBroker) Subscribe(roomID string, handler BrokerHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[roomID] = append(b.handlers[roomID], handler)
	b.logger.Debug("Subscribed local handler to room", zap.String("room_id", roomID))
}

// UnsubscribeRoom unregisters all handlers for a specific room
func (b *WebsocketBroker) UnsubscribeRoom(roomID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.handlers, roomID)
	b.logger.Debug("Unsubscribed all local handlers from room", zap.String("room_id", roomID))
}

func (b *WebsocketBroker) listenRedis() {
	pubsub := b.redisClient.GetClient().Subscribe(b.ctx, b.globalTopic)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-b.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				b.logger.Warn("Redis Pub/Sub channel closed, restarting/falling back")
				b.isFallback = true
				go b.listenInMemory()
				return
			}
			
			var brokerMsg BrokerMessage
			if err := json.Unmarshal([]byte(msg.Payload), &brokerMsg); err != nil {
				b.logger.Error("Failed to unmarshal broker message", zap.Error(err))
				continue
			}

			// Prevent echo loop (skip jika node_id == self)
			if brokerMsg.NodeID == b.nodeID {
				b.logger.Debug("Skipping echo message from self node")
				continue
			}

			b.dispatchEvent(&brokerMsg)
		}
	}
}

func (b *WebsocketBroker) listenInMemory() {
	for {
		select {
		case <-b.ctx.Done():
			return
		case msg := <-b.localBus:
			// In single instance fallback mode, we still deliver but prevent self echo loop
			// depending on usecase, but here we deliver directly. Wait, if it is local fallback,
			// self *is* the only node. If NodeID == self, we still want to dispatch to other local
			// listeners of the room, so we don't drop it. (Skip if not in fallback mode or node_id != self).
			b.dispatchEvent(msg)
		}
	}
}

func (b *WebsocketBroker) dispatchEvent(msg *BrokerMessage) {
	b.mu.RLock()
	handlers, exists := b.handlers[msg.RoomID]
	b.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		return
	}

	for _, handler := range handlers {
		go handler(b.ctx, msg)
	}
}

// Close gracefully shuts down the broker
func (b *WebsocketBroker) Close() error {
	b.mu.Lock()
	if b.isClosed {
		b.mu.Unlock()
		return nil
	}
	b.isClosed = true
	b.mu.Unlock()

	b.cancel()
	b.logger.Info("Websocket broker closed")
	return nil
}
