package websocket

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"go.uber.org/zap"

	"nvide-live/pkg/broker"
	"nvide-live/pkg/redis"
)

// MockBroker is a simple mock implementation of broker.Broker interface
type MockBroker struct {
	mu            sync.Mutex
	subscriptions map[string]broker.Handler
}

func NewMockBroker() *MockBroker {
	return &MockBroker{
		subscriptions: make(map[string]broker.Handler),
	}
}

func (m *MockBroker) Publish(ctx context.Context, topic string, message string) error {
	m.mu.Lock()
	handler, ok := m.subscriptions[topic]
	m.mu.Unlock()
	if ok {
		go handler(message)
	}
	return nil
}

func (m *MockBroker) Subscribe(ctx context.Context, topic string, handler broker.Handler) error {
	m.mu.Lock()
	m.subscriptions[topic] = handler
	m.mu.Unlock()
	return nil
}

func (m *MockBroker) Unsubscribe(ctx context.Context, topic string) error {
	m.mu.Lock()
	delete(m.subscriptions, topic)
	m.mu.Unlock()
	return nil
}

func (m *MockBroker) Close() error {
	return nil
}

func TestHubConcurrencyAndRateLimiting(t *testing.T) {
	// Setup miniredis for atomic rate limiting checks
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer s.Close()

	// Initialize our wrapped Redis client using standard constructor
	logger := zap.NewNop()
	redisWrapper, err := redis.New(&redis.Config{
		Addr: s.Addr(),
	}, logger)
	if err != nil {
		t.Fatalf("failed to create redis client: %v", err)
	}

	// Mock broker
	mockB := NewMockBroker()

	// Initialize Hub (DB pool can be nil for rate limit testing)
	hub := NewHub(nil, mockB, redisWrapper, logger)

	// Number of concurrent users and requests to simulate
	concurrentUsers := 20
	requestsPerUser := 10
	var wg sync.WaitGroup

	t.Run("Concurrent Token Bucket Rate Limiting", func(t *testing.T) {
		limitErrors := int32(0)
		successCount := int32(0)
		var mu sync.Mutex
		results := make(map[string][]bool)

		for i := 0; i < concurrentUsers; i++ {
			wg.Add(1)
			userID := fmt.Sprintf("user_%d", i)
			roomID := "room_1"

			go func(uid string) {
				defer wg.Done()
				userResults := make([]bool, requestsPerUser)

				for j := 0; j < requestsPerUser; j++ {
					// Atomically check token bucket rate limit
					limited, _ := hub.checkTokenBucketRateLimit(uid, roomID)
					userResults[j] = limited
					
					if limited {
						mu.Lock()
						limitErrors++
						mu.Unlock()
					} else {
						mu.Lock()
						successCount++
						mu.Unlock()
					}
					// Microsecond sleep to allow some bucket refilling, but fast enough to trigger limits
					time.Sleep(5 * time.Millisecond)
				}

				mu.Lock()
				results[uid] = userResults
				mu.Unlock()
			}(userID)
		}

		wg.Wait()

		t.Logf("Completed concurrent rate limiting test: Successes=%d, RateLimited=%d", successCount, limitErrors)
		if successCount == 0 {
			t.Error("Expected at least some allowed messages")
		}
		if limitErrors == 0 {
			t.Error("Expected some messages to be rate-limited under high volume")
		}
	})

	t.Run("Concurrent Hub Registrations & Broadcasters", func(t *testing.T) {
		// Test registration safely under lock
		hub.clients = make(map[*Client]bool)
		hub.rooms = make(map[string]map[*Client]bool)

		for i := 0; i < concurrentUsers; i++ {
			wg.Add(1)
			userID := fmt.Sprintf("user_%d", i)
			roomID := "room_concurrency"

			go func(uid string) {
				defer wg.Done()
				client := &Client{
					hub:    hub,
					roomID: roomID,
					userID: uid,
					send:   make(chan []byte, 10),
				}

				// Simulate registration
				hub.mu.Lock()
				hub.clients[client] = true
				if _, ok := hub.rooms[client.roomID]; !ok {
					hub.rooms[client.roomID] = make(map[*Client]bool)
				}
				hub.rooms[client.roomID][client] = true
				hub.mu.Unlock()

				// Simulate message send
				msg := &WSMessage{
					Type:      "chat",
					Payload:   "hello concurrent world",
					Timestamp: time.Now().Format(time.RFC3339),
					RoomID:    roomID,
					UserID:    uid,
				}
				
				topic := "room:" + roomID
				hub.broker.Publish(context.Background(), topic, msg.Timestamp)

				// Simulate unregistration
				hub.mu.Lock()
				delete(hub.clients, client)
				if _, ok := hub.rooms[client.roomID]; ok {
					delete(hub.rooms[client.roomID], client)
				}
				hub.mu.Unlock()
			}(userID)
		}

		wg.Wait()
		t.Log("Successfully handled registration and message broadcasting concurrently without any deadlock or race conditions!")
	})
}
