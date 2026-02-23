package websocket

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type clientConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (c *clientConn) writeJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	return c.conn.WriteJSON(v)
}

type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]map[*clientConn]struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uuid.UUID]map[*clientConn]struct{}),
	}
}

func (h *Hub) Register(userID uuid.UUID, ws *websocket.Conn) *clientConn {
	client := &clientConn{conn: ws}

	h.mu.Lock()
	if _, ok := h.clients[userID]; !ok {
		h.clients[userID] = make(map[*clientConn]struct{})
	}
	h.clients[userID][client] = struct{}{}
	h.mu.Unlock()

	return client
}

func (h *Hub) Unregister(userID uuid.UUID, client *clientConn) {
	h.mu.Lock()
	if set, ok := h.clients[userID]; ok {
		delete(set, client)
		if len(set) == 0 {
			delete(h.clients, userID)
		}
	}
	h.mu.Unlock()
	_ = client.conn.Close()
}

func (h *Hub) SendToUser(userID uuid.UUID, payload any) {
	h.mu.RLock()
	set := h.clients[userID]
	clients := make([]*clientConn, 0, len(set))
	for c := range set {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		if err := c.writeJSON(payload); err != nil {
			h.Unregister(userID, c)
		}
	}
}
