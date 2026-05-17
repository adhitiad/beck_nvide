package websocket

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"nvide-live/pkg/redis"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

type WaitRoomClient struct {
	Hub          *WaitRoomHub
	Conn         *websocket.Conn
	Send         chan []byte
	OccurrenceID string
	UserID       string
	Username     string
	UserLevel    int
}

type WaitRoomWSMessage struct {
	Type         string      `json:"type"`
	Payload      interface{} `json:"payload"`
	Timestamp    string      `json:"timestamp"`
	OccurrenceID string      `json:"occurrence_id,omitempty"`
	UserID       string      `json:"user_id,omitempty"`
	Username     string      `json:"username,omitempty"`
	UserLevel    int         `json:"user_level,omitempty"`
}

type WaitRoomHub struct {
	clients     map[*WaitRoomClient]bool
	rooms       map[string]map[*WaitRoomClient]bool // occurrenceID -> clients
	register    chan *WaitRoomClient
	unregister  chan *WaitRoomClient
	broadcast   chan *WaitRoomWSMessage
	db          *pgxpool.Pool
	redisClient *redis.Client
	logger      *zap.Logger
	mu          sync.RWMutex
}

func NewWaitRoomHub(db *pgxpool.Pool, r *redis.Client, logger *zap.Logger) *WaitRoomHub {
	return &WaitRoomHub{
		clients:     make(map[*WaitRoomClient]bool),
		rooms:       make(map[string]map[*WaitRoomClient]bool),
		register:    make(chan *WaitRoomClient),
		unregister:  make(chan *WaitRoomClient),
		broadcast:   make(chan *WaitRoomWSMessage),
		db:          db,
		redisClient: r,
		logger:      logger,
	}
}

func (h *WaitRoomHub) Run() {
	go h.runCountdownTicker()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			if _, ok := h.rooms[client.OccurrenceID]; !ok {
				h.rooms[client.OccurrenceID] = make(map[*WaitRoomClient]bool)
			}
			h.rooms[client.OccurrenceID][client] = true
			count := len(h.rooms[client.OccurrenceID])
			h.mu.Unlock()

			// Update wait_rooms database count
			go h.updateWaitRoomCount(client.OccurrenceID, count)

			// Emit join event
			h.BroadcastToRoom(client.OccurrenceID, &WaitRoomWSMessage{
				Type: "system",
				Payload: map[string]interface{}{
					"message": fmt.Sprintf("User %s bergabung ke Ruang Tunggu", client.Username),
					"count":   count,
				},
				OccurrenceID: client.OccurrenceID,
			})

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
				if _, ok := h.rooms[client.OccurrenceID]; ok {
					delete(h.rooms[client.OccurrenceID], client)
					if len(h.rooms[client.OccurrenceID]) == 0 {
						delete(h.rooms, client.OccurrenceID)
					}
				}
			}
			h.mu.Unlock()

			h.mu.Lock()
			count := 0
			if _, ok := h.rooms[client.OccurrenceID]; ok {
				count = len(h.rooms[client.OccurrenceID])
			}
			h.mu.Unlock()

			go h.updateWaitRoomCount(client.OccurrenceID, count)

			// Emit leave event
			h.BroadcastToRoom(client.OccurrenceID, &WaitRoomWSMessage{
				Type: "system",
				Payload: map[string]interface{}{
					"message": fmt.Sprintf("User %s meninggalkan Ruang Tunggu", client.Username),
					"count":   count,
				},
				OccurrenceID: client.OccurrenceID,
			})

		case message := <-h.broadcast:
			h.mu.Lock()
			clients := h.rooms[message.OccurrenceID]
			h.mu.Unlock()

			if len(clients) == 0 {
				continue
			}

			// ⚡ STALKER EVENT HUB SWITCH (BOILERPLATE HANDLER)
			// Developer can drop in new event types here to support more interactive waiting room actions.
			switch message.Type {
			case "chat":
				// Rate limit: 1 msg per 5 seconds
				limitKey := fmt.Sprintf("waitroom:rate:%s:%s", message.OccurrenceID, message.UserID)
				exists, _ := h.redisClient.GetClient().Exists(context.Background(), limitKey).Result()
				if exists > 0 {
					h.sendPrivateError(message.UserID, message.OccurrenceID, "RATE_LIMIT", "Anda mengirim pesan terlalu cepat! Jeda slow mode: 5 detik.")
					continue
				}

				// Set Redis key for 5 seconds
				h.redisClient.GetClient().Set(context.Background(), limitKey, "1", 5*time.Second)

				// Save message in Redis volatile list with 1 hour TTL
				chatListKey := fmt.Sprintf("waitroom:chat:%s", message.OccurrenceID)
				msgJSON, _ := json.Marshal(message)
				h.redisClient.GetClient().RPush(context.Background(), chatListKey, msgJSON)
				h.redisClient.GetClient().Expire(context.Background(), chatListKey, 1*time.Hour)

				// Broadcast to room
				h.BroadcastToRoom(message.OccurrenceID, message)

			case "typing":
				// 📝 BOILERPLATE: User typing indicator
				// Anda bisa meneruskan status mengetik user ke penonton lain
				h.BroadcastToRoom(message.OccurrenceID, message)

			case "trivia_answer":
				// 🧠 BOILERPLATE: Kuis Trivia Pra-Siar
				// Integrasikan dengan modul quiz engine atau repositori game di sini
				h.logger.Info("Trivia answer received", zap.String("user", message.Username))

			default:
				// Fallback standard broadcast
				h.BroadcastToRoom(message.OccurrenceID, message)
			}
		}
	}
}

func (h *WaitRoomHub) BroadcastToRoom(occID string, msg *WaitRoomWSMessage) {
	if msg.Timestamp == "" {
		msg.Timestamp = time.Now().Format(time.RFC3339)
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("Failed to marshal wait room WS message", zap.Error(err))
		return
	}

	h.mu.RLock()
	clients, ok := h.rooms[occID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	for client := range clients {
		select {
		case client.Send <- msgBytes:
		default:
			close(client.Send)
			h.mu.Lock()
			delete(h.clients, client)
			delete(h.rooms[occID], client)
			h.mu.Unlock()
		}
	}
}

func (h *WaitRoomHub) sendPrivateError(userID string, occID string, code string, desc string) {
	errMessage := &WaitRoomWSMessage{
		Type: "error",
		Payload: map[string]string{
			"code":        code,
			"description": desc,
		},
		OccurrenceID: occID,
	}

	msgBytes, _ := json.Marshal(errMessage)

	h.mu.RLock()
	clients := h.rooms[occID]
	h.mu.RUnlock()

	for client := range clients {
		if client.UserID == userID {
			select {
			case client.Send <- msgBytes:
			default:
			}
		}
	}
}

func (h *WaitRoomHub) updateWaitRoomCount(occID string, count int) {
	// Find wait room ID or occurrence ID and update count
	ctx := context.Background()
	var exists bool
	_ = h.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM wait_rooms WHERE occurrence_id = $1)", occID).Scan(&exists)
	if exists {
		_, _ = h.db.Exec(ctx, `
			UPDATE wait_rooms
			SET current_user_count = $1,
				peak_user_count = GREATEST(peak_user_count, $1)
			WHERE occurrence_id = $2
		`, count, occID)
	} else {
		// Fetch host ID from occurrence
		var hostID string
		err := h.db.QueryRow(ctx, "SELECT host_id FROM schedule_occurrences WHERE id = $1", occID).Scan(&hostID)
		if err == nil {
			_, _ = h.db.Exec(ctx, `
				INSERT INTO wait_rooms (id, occurrence_id, host_id, status, opened_at, current_user_count, peak_user_count, created_at)
				VALUES (gen_random_uuid(), $1, $2, 'waiting', NOW(), $3, $3, NOW())
			`, occID, hostID, count)
		}
	}
}

func (h *WaitRoomHub) runCountdownTicker() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		h.mu.RLock()
		activeRooms := make([]string, 0, len(h.rooms))
		for roomID := range h.rooms {
			activeRooms = append(activeRooms, roomID)
		}
		h.mu.RUnlock()

		for _, occID := range activeRooms {
			h.mu.RLock()
			clients := h.rooms[occID]
			h.mu.RUnlock()

			if len(clients) == 0 {
				continue
			}

			var startAt time.Time
			err := h.db.QueryRow(context.Background(), "SELECT occurrence_start_at FROM schedule_occurrences WHERE id = $1", occID).Scan(&startAt)
			if err != nil {
				continue
			}

			remaining := int(time.Until(startAt).Seconds())
			if remaining < -600 {
				// Missed for more than 10 mins: auto close wait room
				h.BroadcastToRoom(occID, &WaitRoomWSMessage{
					Type: "host_missed",
					Payload: map[string]interface{}{
						"message": "Host tidak memulai siaran sesuai jadwal. Ruang tunggu ditutup.",
					},
					OccurrenceID: occID,
				})

				// Force close all client connections gracefully
				h.mu.Lock()
				for client := range clients {
					client.Conn.Close()
					delete(h.clients, client)
				}
				delete(h.rooms, occID)
				h.mu.Unlock()
				continue
			}

			if remaining < 0 {
				remaining = 0
			}

			h.BroadcastToRoom(occID, &WaitRoomWSMessage{
				Type: "countdown",
				Payload: map[string]interface{}{
					"seconds_remaining": remaining,
				},
				OccurrenceID: occID,
			})
		}
	}
}

// ServeWaitRoomWS upgrades HTTP request to Wait Room WebSocket Client
func (h *WaitRoomHub) ServeWaitRoomWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	occID := vars["occurrence_id"]
	if occID == "" {
		occID = r.URL.Query().Get("occurrence_id")
	}
	userID := r.URL.Query().Get("user_id")

	if occID == "" || userID == "" {
		http.Error(w, "occurrence_id and user_id are required", http.StatusBadRequest)
		return
	}

	// Validate occurrence exists and is upcoming or live
	var status string
	var occurrenceStartAt time.Time
	err := h.db.QueryRow(context.Background(), "SELECT status, occurrence_start_at FROM schedule_occurrences WHERE id = $1", occID).Scan(&status, &occurrenceStartAt)
	if err != nil {
		http.Error(w, "Occurrence not found", http.StatusNotFound)
		return
	}

	// Reject if closed or cancelled
	if status == "cancelled" || status == "completed" || status == "missed" {
		http.Error(w, "Occurrence is closed or cancelled", http.StatusBadRequest)
		return
	}

	// Soft Limit: Max 500 users per wait room
	h.mu.RLock()
	currentCount := len(h.rooms[occID])
	h.mu.RUnlock()
	if currentCount >= 500 {
		http.Error(w, "Wait room is full (max 500 users). Please try again later.", http.StatusServiceUnavailable)
		return
	}

	// Load user details
	var username string
	var userLevel int
	err = h.db.QueryRow(context.Background(), "SELECT username, user_level FROM users WHERE id = $1", userID).Scan(&username, &userLevel)
	if err != nil {
		username = "Gamu-Guest"
		userLevel = 1
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade wait room websocket connection", zap.Error(err))
		return
	}

	client := &WaitRoomClient{
		Hub:          h,
		Conn:         conn,
		Send:         make(chan []byte, 256),
		OccurrenceID: occID,
		UserID:       userID,
		Username:     username,
		UserLevel:    userLevel,
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *WaitRoomClient) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()
	c.Conn.SetReadLimit(5120)
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second)); return nil })
	for {
		_, msgBytes, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
		msgBytes = bytes.TrimSpace(bytes.Replace(msgBytes, []byte{'\n'}, []byte{' '}, -1))

		var wsMsg WaitRoomWSMessage
		if err := json.Unmarshal(msgBytes, &wsMsg); err != nil {
			continue
		}

		// Inject metadata
		wsMsg.OccurrenceID = c.OccurrenceID
		wsMsg.UserID = c.UserID
		wsMsg.Username = c.Username
		wsMsg.UserLevel = c.UserLevel

		c.Hub.broadcast <- &wsMsg
	}
}

func (c *WaitRoomClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
