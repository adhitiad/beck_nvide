package broker

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"nvide-live/pkg/redis"
)

// Handler represents a message handler function
type Handler func(message string)

// Broker defines the message broker interface
type Broker interface {
	Publish(ctx context.Context, topic string, message string) error
	Subscribe(ctx context.Context, topic string, handler Handler) error
	Close() error
}

// HybridBroker implements Broker using Redis with in-memory fallback
type HybridBroker struct {
	redisClient *redis.Client
	logger      *zap.Logger

	// In-memory fallback
	mu       sync.RWMutex
	handlers map[string][]Handler
}

// NewHybridBroker creates a new hybrid broker
func NewHybridBroker(redisClient *redis.Client, logger *zap.Logger) Broker {
	return &HybridBroker{
		redisClient: redisClient,
		logger:      logger,
		handlers:    make(map[string][]Handler),
	}
}

// Publish publishes a message to a topic
func (b *HybridBroker) Publish(ctx context.Context, topic string, message string) error {
	var err error
	if b.redisClient != nil && b.redisClient.Health(ctx) == nil {
		// Use Redis
		err = b.redisClient.GetClient().Publish(ctx, topic, message).Err()
	} else {
		// Fallback to in-memory
		err = b.publishInMemory(topic, message)
	}

	if err != nil {
		b.logger.Error("Failed to publish message", zap.String("topic", topic), zap.Error(err))
	}
	return err
}

// Subscribe subscribes to a topic
func (b *HybridBroker) Subscribe(ctx context.Context, topic string, handler Handler) error {
	if b.redisClient != nil && b.redisClient.Health(ctx) == nil {
		// Use Redis
		pubsub := b.redisClient.GetClient().Subscribe(ctx, topic)

		// Run a goroutine to listen for messages
		go func() {
			defer pubsub.Close()
			ch := pubsub.Channel()
			for msg := range ch {
				handler(msg.Payload)
			}
		}()

		b.logger.Info("Subscribed to Redis topic", zap.String("topic", topic))
	} else {
		// Fallback to in-memory
		b.subscribeInMemory(topic, handler)
		b.logger.Info("Subscribed to in-memory topic", zap.String("topic", topic))
	}
	return nil
}

// Close closes the broker
func (b *HybridBroker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = make(map[string][]Handler)
	return nil
}

func (b *HybridBroker) publishInMemory(topic string, message string) error {
	b.mu.RLock()
	handlers := b.handlers[topic]
	b.mu.RUnlock()

	for _, handler := range handlers {
		go handler(message)
	}
	return nil
}

func (b *HybridBroker) subscribeInMemory(topic string, handler Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[topic] = append(b.handlers[topic], handler)
}
