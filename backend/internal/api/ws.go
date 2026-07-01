package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

// Hub tracks connected WebSocket clients and fans out broadcast messages. It is
// the realtime backbone for the unified inbox and interaction updates (Tier 3).
type Hub struct {
	mu      sync.RWMutex
	clients map[*wsClient]struct{}
}

// NewHub returns an empty hub.
func NewHub() *Hub {
	return &Hub{clients: make(map[*wsClient]struct{})}
}

// Broadcast sends msg to every connected client. Slow clients that cannot keep
// up are dropped rather than blocking the broadcast.
func (h *Hub) Broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.send <- msg:
		default:
			// Client is backed up; close its send channel to disconnect it.
			close(c.send)
			delete(h.clients, c)
		}
	}
}

// Count returns the number of connected clients.
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) add(c *wsClient) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) remove(c *wsClient) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
}

type wsClient struct {
	conn   *websocket.Conn
	send   chan []byte
	userID string
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Same-origin by default; cross-origin policy is tightened in Tier 6.
}

// handleWS upgrades the connection and registers a client. It requires an
// authenticated user (enforced by RequireAuth on the route).
func (a *API) handleWS(w http.ResponseWriter, r *http.Request) {
	u, ok := currentUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade already wrote an error response.
		return
	}
	client := &wsClient{conn: conn, send: make(chan []byte, 16), userID: u}
	a.hub.add(client)
	a.log.Debug("ws connected", "user", u, "clients", a.hub.Count())

	go a.writePump(client)
	a.readPump(client)
}

// readPump drains inbound frames (and keeps the connection alive via pongs)
// until the client disconnects, then unregisters it.
func (a *API) readPump(c *wsClient) {
	defer func() {
		a.hub.remove(c)
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(1 << 20)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
		// Inbound client messages are ignored for now; realtime is
		// server-to-client push in Tier 1.
	}
}

// writePump delivers queued messages and periodic pings to the client.
func (a *API) writePump(c *wsClient) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
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
