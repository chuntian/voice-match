// Package signal implements the WebSocket signaling server: connection
// management (Hub), call state machine (CallManager), and HTTP handler
// (handler.go).
package signal

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// writeWait is the time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// pongWait is the time allowed to read the next pong message.
	// The server also enforces its own heartbeat timeout (30s).
	pongWait = 60 * time.Second

	// pingPeriod sends pings to the peer within pongWait. Must be
	// less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// heartbeatTimeout disconnects the client if no heartbeat is
	// received within this duration.
	heartbeatTimeout = 30 * time.Second

	// maxMessageSize is the maximum allowed message size.
	maxMessageSize = 64 * 1024
)

// Client wraps a single WebSocket connection.
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	userID   string
	deviceID string
	send     chan []byte

	mu             sync.Mutex
	lastHeartbeat  time.Time
	authenticated  bool
}

// Hub maintains the set of active clients and routes messages.
type Hub struct {
	mu      sync.RWMutex
	clients map[string]*Client // userID -> *Client

	// onMessage is invoked for each inbound message after authentication.
	onMessage func(userID string, raw []byte)
}

// NewHub creates a new Hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]*Client),
	}
}

// SetMessageHandler registers the inbound message handler.
func (h *Hub) SetMessageHandler(fn func(userID string, raw []byte)) {
	h.onMessage = fn
}

// Register adds a client to the hub. If a client with the same userID
// already exists, the old connection is closed (single-device login
// model; multi-device can be added later by keying on userID:deviceID).
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if existing, ok := h.clients[c.userID]; ok {
		close(existing.send)
		_ = existing.conn.Close()
	}
	h.clients[c.userID] = c
	log.Printf("[hub] registered user=%s device=%s total=%d", c.userID, c.deviceID, len(h.clients))
}

// Unregister removes a client from the hub and closes its connection.
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if current, ok := h.clients[c.userID]; ok && current == c {
		delete(h.clients, c.userID)
		close(c.send)
		_ = c.conn.Close()
		log.Printf("[hub] unregistered user=%s total=%d", c.userID, len(h.clients))
	}
}

// SendToUser sends a raw JSON message to the user with the given userID.
// Returns false if the user is not online.
func (h *Hub) SendToUser(userID string, data []byte) bool {
	h.mu.RLock()
	c, ok := h.clients[userID]
	h.mu.RUnlock()
	if !ok {
		return false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case c.send <- data:
		return true
	default:
		// send buffer full; close connection
		go h.Unregister(c)
		return false
	}
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(data []byte) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for _, c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		c.mu.Lock()
		select {
		case c.send <- data:
		default:
			go h.Unregister(c)
		}
		c.mu.Unlock()
	}
}

// IsOnline returns true if the given user has an active connection.
func (h *Hub) IsOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[userID]
	return ok
}

// OnlineCount returns the number of currently connected clients.
func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// MarkAuthenticated marks the client as having completed the hello handshake.
func (c *Client) MarkAuthenticated() {
	c.mu.Lock()
	c.authenticated = true
	c.lastHeartbeat = time.Now()
	c.mu.Unlock()
}

// UpdateHeartbeat records the latest heartbeat timestamp.
func (c *Client) UpdateHeartbeat(ts int64) {
	c.mu.Lock()
	c.lastHeartbeat = time.Now()
	c.mu.Unlock()
}

// readPump pumps messages from the WebSocket connection to the hub.
// It runs in its own goroutine for each connection.
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[hub] read error user=%s: %v", c.userID, err)
			}
			break
		}

		c.mu.Lock()
		authed := c.authenticated
		c.mu.Unlock()

		if !authed {
			// The first message must be a hello; the handler will call
			// MarkAuthenticated if it succeeds. We still dispatch to
			// onMessage so the handler can validate and respond.
		}

		if c.hub.onMessage != nil {
			c.hub.onMessage(c.userID, data)
		}
	}
}

// writePump pumps messages from the hub to the WebSocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			if _, err := w.Write(message); err != nil {
				return
			}

			// Flush queued messages in the same write.
			n := len(c.send)
			for i := 0; i < n; i++ {
				extra := <-c.send
				if _, err := w.Write(extra); err != nil {
					return
				}
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// heartbeatWatcher periodically checks that each client sends heartbeats.
// It should be run in a single goroutine.
func (h *Hub) heartbeatWatcher() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		h.mu.RLock()
		toEvict := make([]*Client, 0)
		for _, c := range h.clients {
			c.mu.Lock()
			if c.authenticated && now.Sub(c.lastHeartbeat) > heartbeatTimeout {
				toEvict = append(toEvict, c)
			}
			c.mu.Unlock()
		}
		h.mu.RUnlock()

		for _, c := range toEvict {
			log.Printf("[hub] heartbeat timeout, disconnecting user=%s", c.userID)
			h.Unregister(c)
		}
	}
}

// StartHeartbeatWatcher launches the background heartbeat goroutine.
func (h *Hub) StartHeartbeatWatcher() {
	go h.heartbeatWatcher()
}

// NewClient creates a new Client.
func NewClient(hub *Hub, conn *websocket.Conn, userID, deviceID string) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		userID:   userID,
		deviceID: deviceID,
		send:     make(chan []byte, 64),
	}
}

// UserID returns the client's user ID.
func (c *Client) UserID() string { return c.userID }

// DeviceID returns the client's device ID.
func (c *Client) DeviceID() string { return c.deviceID }

// sendJSON marshals v and writes it to the client's send channel.
func (c *Client) sendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	select {
	case c.send <- data:
		return nil
	default:
		return errSendBufferFull
	}
}

// errSendBufferFull is returned when the client's send buffer is full.
var errSendBufferFull = &clientError{"send buffer full"}

type clientError struct{ msg string }

func (e *clientError) Error() string { return e.msg }
