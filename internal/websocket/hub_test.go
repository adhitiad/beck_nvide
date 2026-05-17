package websocket

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"
	"nvide-live/pkg/broker"
)

type mockBroker struct {
	PublishedMessages []string
}

func (m *mockBroker) Publish(ctx context.Context, topic string, message string) error {
	m.PublishedMessages = append(m.PublishedMessages, message)
	return nil
}

func (m *mockBroker) Subscribe(ctx context.Context, topic string, handler broker.Handler) error {
	return nil
}

func (m *mockBroker) Close() error {
	return nil
}

func TestHub_RegisterAndUnregister(t *testing.T) {
	logger := zap.NewNop()
	broker := &mockBroker{}

	hub := NewHub(nil, broker, nil, logger)
	go hub.Run()

	client := &Client{
		hub:    hub,
		send:   make(chan []byte, 256),
		roomID: "room-1",
		userID: "user-1",
	}

	// Test Register
	hub.register <- client

	// Give it some time to process
	time.Sleep(100 * time.Millisecond)

	hub.mu.RLock()
	if !hub.clients[client] {
		t.Errorf("Client was not registered")
	}
	if !hub.rooms["room-1"][client] {
		t.Errorf("Client was not added to room")
	}
	hub.mu.RUnlock()

	// Test Unregister
	hub.unregister <- client

	time.Sleep(100 * time.Millisecond)

	hub.mu.RLock()
	if hub.clients[client] {
		t.Errorf("Client was not unregistered")
	}
	if len(hub.rooms["room-1"]) != 0 {
		t.Errorf("Room was not cleaned up")
	}
	hub.mu.RUnlock()
}
