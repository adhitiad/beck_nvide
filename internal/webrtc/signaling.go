package webrtc

import (
	"encoding/json"
	"sync"

	"github.com/pion/webrtc/v4"
	"go.uber.org/zap"
)

// SignalingMessage represents a WebRTC signaling message
type SignalingMessage struct {
	Type   string          `json:"type"` // offer, answer, ice_candidate
	PeerID string          `json:"peer_id"`
	Data   json.RawMessage `json:"data"`
}

// PeerState represents the state of a peer
type PeerState struct {
	ID             string
	IsHost         bool
	Connection     *webrtc.PeerConnection
	PendingICE     []*webrtc.ICECandidateInit
	RemoteDescSet  bool
	SignalSendChan chan SignalingMessage
	mu             sync.Mutex
}

// Room represents a streaming room
type Room struct {
	ID      string
	Host    *PeerState
	Viewers map[string]*PeerState
	Tracks  map[string]*webrtc.TrackLocalStaticRTP // the tracks from the host (kind -> track)
	mu      sync.RWMutex
}

// RoomManager manages WebRTC rooms and peers
type RoomManager struct {
	rooms  map[string]*Room
	logger *zap.Logger
	mu     sync.RWMutex
}

// NewRoomManager creates a new room manager
func NewRoomManager(logger *zap.Logger) *RoomManager {
	return &RoomManager{
		rooms:  make(map[string]*Room),
		logger: logger,
	}
}

// GetOrCreateRoom gets or creates a room
func (m *RoomManager) GetOrCreateRoom(roomID string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()

	room, exists := m.rooms[roomID]
	if !exists {
		room = &Room{
			ID:      roomID,
			Viewers: make(map[string]*PeerState),
			Tracks:  make(map[string]*webrtc.TrackLocalStaticRTP),
		}
		m.rooms[roomID] = room
	}
	return room
}

// RemoveRoom removes a room
func (m *RoomManager) RemoveRoom(roomID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if room, exists := m.rooms[roomID]; exists {
		room.mu.Lock()
		if room.Host != nil && room.Host.Connection != nil {
			room.Host.Connection.Close()
		}
		for _, viewer := range room.Viewers {
			if viewer.Connection != nil {
				viewer.Connection.Close()
			}
		}
		room.mu.Unlock()
		delete(m.rooms, roomID)
	}
}

// HandleHostConnection sets up the host PeerConnection
func (m *RoomManager) HandleHostConnection(roomID, peerID string, sendChan chan SignalingMessage) (*PeerState, error) {
	room := m.GetOrCreateRoom(roomID)
	room.mu.Lock()
	defer room.mu.Unlock()

	// If host already exists, clean it up
	if room.Host != nil && room.Host.Connection != nil {
		room.Host.Connection.Close()
	}

	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	hostPeer := &PeerState{
		ID:             peerID,
		IsHost:         true,
		Connection:     peerConnection,
		SignalSendChan: sendChan,
	}
	room.Host = hostPeer

	// Accept tracks (audio and video)
	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		kind := track.Kind().String()
		m.logger.Info("Received host track", zap.String("kind", kind))

		// Create a local track to be relayed to viewers
		trackLocal, err := webrtc.NewTrackLocalStaticRTP(track.Codec().RTPCodecCapability, kind, "pion")
		if err != nil {
			m.logger.Error("Failed to create local track", zap.Error(err), zap.String("kind", kind))
			return
		}

		room.mu.Lock()
		room.Tracks[kind] = trackLocal
		// Automatically add this new track to existing viewers
		for _, viewer := range room.Viewers {
			if viewer.Connection != nil {
				if _, err := viewer.Connection.AddTrack(trackLocal); err != nil {
					m.logger.Error("Failed to add new track to existing viewer", zap.Error(err), zap.String("peer_id", viewer.ID))
				}
			}
		}
		room.mu.Unlock()

		// Read RTP packets from the host and write to the local track
		go func() {
			rtpBuf := make([]byte, 1400)
			for {
				i, _, readErr := track.Read(rtpBuf)
				if readErr != nil {
					m.logger.Warn("Host track read error", zap.Error(readErr), zap.String("kind", kind))
					return
				}

				if _, writeErr := trackLocal.Write(rtpBuf[:i]); writeErr != nil {
					// Only log if it's not a closed connection error
					if writeErr.Error() != "io: read/write on closed pipe" {
						m.logger.Debug("TrackLocal write error", zap.Error(writeErr), zap.String("kind", kind))
					}
				}
			}
		}()
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		data, _ := json.Marshal(candidate.ToJSON())
		sendChan <- SignalingMessage{
			Type:   "ice_candidate",
			PeerID: peerID,
			Data:   data,
		}
	})

	return hostPeer, nil
}

// HandleViewerConnection sets up a viewer PeerConnection
func (m *RoomManager) HandleViewerConnection(roomID, peerID string, sendChan chan SignalingMessage) (*PeerState, error) {
	room := m.GetOrCreateRoom(roomID)

	peerConnection, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	viewerPeer := &PeerState{
		ID:             peerID,
		IsHost:         false,
		Connection:     peerConnection,
		SignalSendChan: sendChan,
	}

	room.mu.Lock()
	room.Viewers[peerID] = viewerPeer
	tracks := make([]*webrtc.TrackLocalStaticRTP, 0, len(room.Tracks))
	for _, t := range room.Tracks {
		tracks = append(tracks, t)
	}
	room.mu.Unlock()

	// Add all available tracks to viewer
	for _, trackLocal := range tracks {
		if _, err = peerConnection.AddTrack(trackLocal); err != nil {
			m.logger.Error("Failed to add track to viewer", zap.Error(err), zap.String("peer_id", peerID))
		}
	}

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		data, _ := json.Marshal(candidate.ToJSON())
		sendChan <- SignalingMessage{
			Type:   "ice_candidate",
			PeerID: peerID,
			Data:   data,
		}
	})

	return viewerPeer, nil
}

// RemovePeer cleans up a peer
func (m *RoomManager) RemovePeer(roomID, peerID string) {
	room := m.GetOrCreateRoom(roomID)
	room.mu.Lock()
	defer room.mu.Unlock()

	if room.Host != nil && room.Host.ID == peerID {
		room.Host.Connection.Close()
		room.Host = nil
		// Maybe close room?
	} else if viewer, exists := room.Viewers[peerID]; exists {
		viewer.Connection.Close()
		delete(room.Viewers, peerID)
	}
}

// ProcessSignaling handles incoming signaling messages
func (m *RoomManager) ProcessSignaling(roomID, peerID string, msg SignalingMessage) error {
	room := m.GetOrCreateRoom(roomID)
	room.mu.RLock()
	var peer *PeerState
	if room.Host != nil && room.Host.ID == peerID {
		peer = room.Host
	} else if viewer, exists := room.Viewers[peerID]; exists {
		peer = viewer
	}
	room.mu.RUnlock()

	if peer == nil {
		m.logger.Warn("Peer not found for signaling", zap.String("peer_id", peerID))
		return nil
	}

	switch msg.Type {
	case "offer":
		var offer webrtc.SessionDescription
		if err := json.Unmarshal(msg.Data, &offer); err != nil {
			return err
		}
		if err := peer.Connection.SetRemoteDescription(offer); err != nil {
			return err
		}
		peer.mu.Lock()
		peer.RemoteDescSet = true
		pendingICE := peer.PendingICE
		peer.PendingICE = nil
		peer.mu.Unlock()

		// Flush pending ICE
		for _, ice := range pendingICE {
			if err := peer.Connection.AddICECandidate(*ice); err != nil {
				m.logger.Error("Failed to add ICE candidate", zap.Error(err))
			}
		}

		// Create Answer
		answer, err := peer.Connection.CreateAnswer(nil)
		if err != nil {
			return err
		}
		if err = peer.Connection.SetLocalDescription(answer); err != nil {
			return err
		}

		data, _ := json.Marshal(answer)
		peer.SignalSendChan <- SignalingMessage{
			Type:   "answer",
			PeerID: peerID,
			Data:   data,
		}

	case "answer":
		var answer webrtc.SessionDescription
		if err := json.Unmarshal(msg.Data, &answer); err != nil {
			return err
		}
		if err := peer.Connection.SetRemoteDescription(answer); err != nil {
			return err
		}
		peer.mu.Lock()
		peer.RemoteDescSet = true
		pendingICE := peer.PendingICE
		peer.PendingICE = nil
		peer.mu.Unlock()

		// Flush pending ICE
		for _, ice := range pendingICE {
			if err := peer.Connection.AddICECandidate(*ice); err != nil {
				m.logger.Error("Failed to add ICE candidate", zap.Error(err))
			}
		}

	case "ice_candidate":
		var ice webrtc.ICECandidateInit
		if err := json.Unmarshal(msg.Data, &ice); err != nil {
			return err
		}

		peer.mu.Lock()
		if peer.RemoteDescSet {
			peer.mu.Unlock()
			if err := peer.Connection.AddICECandidate(ice); err != nil {
				m.logger.Error("Failed to add ICE candidate", zap.Error(err))
			}
		} else {
			peer.PendingICE = append(peer.PendingICE, &ice)
			peer.mu.Unlock()
		}
	}
	return nil
}
