package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"backend/internal/models"
	"backend/internal/repository"

	"github.com/gofiber/websocket/v2"
	"github.com/redis/go-redis/v9"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Heartbeat interval for version updates (prevents request storm)
	// Frontend only fetches when version changes, max once per heartbeat
	versionHeartbeatInterval = 2 * time.Second

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// Client represents a WebSocket client connection
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// Hub maintains the set of active clients and broadcasts messages to them
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Redis repository for fetching leaderboard data
	redisRepo *repository.RedisRepository

	// Redis client for pub/sub (deprecated - using version polling instead)
	redisClient *redis.Client

	// Mutex for thread-safe operations
	mu sync.RWMutex
	
	// Last known version for change detection
	lastVersion int64
}

// VersionUpdate represents the version heartbeat message
type VersionUpdate struct {
	Type    string `json:"type"`
	Version int64  `json:"version"`
}

// LeaderboardUpdate represents the data structure sent to WebSocket clients (deprecated)
type LeaderboardUpdate struct {
	Type      string                     `json:"type"`
	Timestamp string                     `json:"timestamp"`
	Data      []models.LeaderboardEntry  `json:"data"`
	Total     int                        `json:"total"`
}

// NewHub creates a new WebSocket hub
func NewHub(redisRepo *repository.RedisRepository, redisClient *redis.Client) *Hub {
	return &Hub{
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		clients:     make(map[*Client]bool),
		redisRepo:   redisRepo,
		redisClient: redisClient,
		lastVersion: 0,
	}
}

// Run starts the WebSocket hub
func (h *Hub) Run(ctx context.Context) {
	log.Println("🚀 WebSocket Hub started")

	// Ticker to check version every 2 seconds
	versionTicker := time.NewTicker(versionHeartbeatInterval)
	defer versionTicker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("✅ Client connected (Total: %d)", len(h.clients))

			// Send initial version to new client
			h.sendInitialVersion(client)

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			log.Printf("❌ Client disconnected (Total: %d)", len(h.clients))

		case <-versionTicker.C:
			// Check if version changed and broadcast if necessary
			h.checkAndBroadcastVersion(ctx)

		case <-ctx.Done():
			log.Println("🛑 WebSocket Hub shutting down")
			return
		}
	}
}

// checkAndBroadcastVersion checks if the version has changed and broadcasts to all clients
func (h *Hub) checkAndBroadcastVersion(ctx context.Context) {
	currentVersion, err := h.redisRepo.GetLeaderboardVersion(ctx)
	if err != nil {
		log.Printf("❌ Failed to get leaderboard version: %v", err)
		return
	}

	// Only broadcast if version has changed
	if currentVersion != h.lastVersion {
		h.lastVersion = currentVersion
		log.Printf("📡 Version changed to %d, broadcasting to clients", currentVersion)

		// Create version update message
		update := VersionUpdate{
			Type:    "VERSION_UPDATE",
			Version: currentVersion,
		}

		// Marshal to JSON
		message, err := json.Marshal(update)
		if err != nil {
			log.Printf("❌ Failed to marshal version update: %v", err)
			return
		}

		// Broadcast to all connected clients
		h.mu.RLock()
		for client := range h.clients {
			select {
			case client.send <- message:
			default:
				// Client's send buffer is full, skip this client
				log.Printf("⚠️ Client send buffer full, skipping")
			}
		}
		h.mu.RUnlock()
	}
}

// sendInitialVersion sends the current version to a newly connected client
func (h *Hub) sendInitialVersion(client *Client) {
	ctx := context.Background()

	currentVersion, err := h.redisRepo.GetLeaderboardVersion(ctx)
	if err != nil {
		log.Printf("❌ Failed to get initial version: %v", err)
		return
	}

	// Update lastVersion if this is the first client
	if h.lastVersion == 0 {
		h.lastVersion = currentVersion
	}

	// Create version update message
	update := VersionUpdate{
		Type:    "VERSION_UPDATE",
		Version: currentVersion,
	}

	// Marshal to JSON
	message, err := json.Marshal(update)
	if err != nil {
		log.Printf("❌ Failed to marshal initial version: %v", err)
		return
	}

	// Check if client is still registered before sending
	h.mu.RLock()
	_, exists := h.clients[client]
	h.mu.RUnlock()

	if !exists {
		log.Println("⚠️ Client disconnected before initial version could be sent")
		return
	}

	// Send to client with timeout to prevent blocking
	select {
	case client.send <- message:
		log.Printf("✅ Sent initial version (%d) to new client", currentVersion)
	case <-time.After(2 * time.Second):
		log.Println("⚠️ Timeout sending initial version - client may be slow")
	}
}

// GetClientCount returns the current number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// readPump pumps messages from the WebSocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	// Don't set read deadline or limits for browser WebSocket clients
	// Browser WebSockets handle ping/pong automatically at the protocol level
	// and don't expose pong handling to JavaScript
	
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			// Client disconnected or error occurred
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("⚠️ WebSocket unexpected close: %v", err)
			}
			break
		}
		// We don't expect messages from clients, but if we receive any, just ignore them
	}
}

// writePump pumps messages from the hub to the WebSocket connection
func (c *Client) writePump() {
	// Don't use ping ticker for browser clients
	// Browser WebSockets handle keepalive at the protocol level
	defer func() {
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}
		}
	}
}

// ServeWS handles WebSocket requests from clients
func ServeWS(hub *Hub, conn *websocket.Conn) {
	client := &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
	
	client.hub.register <- client

	// Start write pump in goroutine
	go client.writePump()
	
	// Run read pump in current goroutine (blocks until disconnect)
	client.readPump()
}
