package webrtc

import (
	"encoding/json"
	"testing"

	"go.uber.org/zap"
)

func TestRoomManager_HostConnection(t *testing.T) {
	// Not full integration test, just testing the RoomManager methods
	logger, _ := zap.NewDevelopment()
	m := NewRoomManager(logger)

	roomID := "room1"
	hostID := "host1"
	sendChan := make(chan SignalingMessage, 10)

	peer, err := m.HandleHostConnection(roomID, hostID, sendChan)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if peer == nil {
		t.Fatal("Expected peer state, got nil")
	}
	if !peer.IsHost {
		t.Error("Expected IsHost to be true")
	}

	room := m.GetOrCreateRoom(roomID)
	if room.Host == nil || room.Host.ID != hostID {
		t.Error("Room does not have the correct host set")
	}

	m.RemoveRoom(roomID)
}

func TestRoomManager_ViewerConnection(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	m := NewRoomManager(logger)

	roomID := "room1"
	viewerID := "viewer1"
	sendChan := make(chan SignalingMessage, 10)

	peer, err := m.HandleViewerConnection(roomID, viewerID, sendChan)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if peer == nil {
		t.Fatal("Expected peer state, got nil")
	}
	if peer.IsHost {
		t.Error("Expected IsHost to be false")
	}

	room := m.GetOrCreateRoom(roomID)
	if _, exists := room.Viewers[viewerID]; !exists {
		t.Error("Viewer was not added to the room")
	}

	m.RemovePeer(roomID, viewerID)
	if _, exists := room.Viewers[viewerID]; exists {
		t.Error("Viewer was not removed from the room")
	}
}

func TestRoomManager_ProcessSignaling(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	m := NewRoomManager(logger)

	roomID := "room1"
	hostID := "host1"
	sendChan := make(chan SignalingMessage, 10)

	_, _ = m.HandleHostConnection(roomID, hostID, sendChan)

	// Since we mock the actual Pion connection setup, parsing offer might fail if invalid SDP
	// Just passing an invalid json should return an error
	err := m.ProcessSignaling(roomID, hostID, SignalingMessage{
		Type: "offer",
		Data: json.RawMessage(`{invalid`),
	})
	if err == nil {
		t.Error("Expected error on invalid JSON, got nil")
	}
}
