package hub

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Hub broadcasts messages to all connected viewer websockets.
type Hub struct {
	mu         sync.Mutex
	clients    map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
}

// Client is one viewer websocket connection.
type Client struct {
	Conn *websocket.Conn
	Send chan []byte
}

// New starts a Hub and its dispatch loop.
func New() *Hub {
	h := &Hub{
		clients:    make(map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
	}
	go h.run()
	return h
}

func (h *Hub) run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()
		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.Send)
			}
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.Lock()
			for c := range h.clients {
				select {
				case c.Send <- msg:
				default: // slow consumer: drop rather than block the hub
				}
			}
			h.mu.Unlock()
		}
	}
}

// Register subscribes a client to broadcasts.
func (h *Hub) Register(c *Client) { h.register <- c }

// Unregister removes a client; its Send channel is closed.
func (h *Hub) Unregister(c *Client) { h.unregister <- c }

// Broadcast delivers msg to every connected client.
func (h *Hub) Broadcast(msg []byte) { h.broadcast <- msg }
