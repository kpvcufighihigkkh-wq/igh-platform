package ws

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/igh-platform/igh-platform/internal/shared/logger"
	"github.com/igh-platform/igh-platform/internal/shared/types"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	topics map[string]bool
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan types.WSMessage
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan types.WSMessage, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				if client.matchTopic(msg.Topic) {
					select {
					case client.send <- mustMarshal(msg):
					default:
						go func(c *Client) { h.unregister <- c }(client)
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Broadcast(topic string, data any) {
	h.broadcast <- types.WSMessage{
		Topic:     topic,
		Timestamp: time.Now(),
		Data:      data,
	}
}

func (h *Hub) HandleConnect(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.L.Errorw("ws upgrade failed", "error", err)
		return
	}

	topics := parseTopics(r.URL.Query().Get("subscribe"))
	client := &Client{
		hub:    h,
		conn:   conn,
		send:   make(chan []byte, 64),
		topics: topics,
	}

	h.register <- client
	go client.writePump()
	go client.readPump()
}

func (c *Client) matchTopic(topic string) bool {
	if len(c.topics) == 0 {
		return true
	}
	for t := range c.topics {
		if t == topic || (len(t) > 0 && t[len(t)-1] == '*' && len(topic) >= len(t)-1 && topic[:len(t)-1] == t[:len(t)-1]) {
			return true
		}
	}
	return false
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
	}
}

func parseTopics(s string) map[string]bool {
	topics := make(map[string]bool)
	if s == "" {
		return topics
	}
	for _, t := range splitComma(s) {
		if t != "" {
			topics[t+".*"] = true
		}
	}
	return topics
}

func splitComma(s string) []string {
	var result []string
	start := 0
	for i := range s {
		if s[i] == ',' {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
