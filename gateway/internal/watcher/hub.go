package watcher

import (
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// Client represents a WebSocket client
type Client struct {
	UserID string
	Conn   *websocket.Conn
	Send   chan Notification
}

// Hub maintains active clients and broadcasts notifications
type Hub struct {
	clients    map[string][]*Client // userID -> clients
	register   chan *Client
	unregister chan *Client
	broadcast  chan Notification
	mutex      sync.RWMutex
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string][]*Client),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Notification),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case notification := <-h.broadcast:
			h.broadcastToUser(notification)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if h.clients[client.UserID] == nil {
		h.clients[client.UserID] = make([]*Client, 0)
	}
	h.clients[client.UserID] = append(h.clients[client.UserID], client)
	log.Printf("Client registered for user %s. Total clients for user: %d",
		client.UserID, len(h.clients[client.UserID]))

	go h.writePump(client)
}

func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if clients, exists := h.clients[client.UserID]; exists {
		for i, c := range clients {
			if c == client {
				// Remove client from slice
				h.clients[client.UserID] = append(clients[:i], clients[i+1:]...)
				close(client.Send)
				client.Conn.Close()
				break
			}
		}

		// Clean up empty user entry
		if len(h.clients[client.UserID]) == 0 {
			delete(h.clients, client.UserID)
		}

		log.Printf("Client unregistered for user %s", client.UserID)
	}
}

func (h *Hub) broadcastToUser(notification Notification) {
	h.mutex.RLock()
	clients := h.clients[notification.UserID]
	h.mutex.RUnlock()

	for _, client := range clients {
		select {
		case client.Send <- notification:
		default:
			// Client's send channel is full, disconnect them
			h.unregister <- client
		}
	}
}

func (h *Hub) writePump(client *Client) {
	defer func() {
		h.unregister <- client
	}()

	for notification := range client.Send {
		if err := client.Conn.WriteJSON(notification); err != nil {
			log.Printf("WebSocket write error: %v", err)
			return
		}
	}
}

// SendNotification sends a notification to the hub for broadcasting
func (h *Hub) SendNotification(notification Notification) {
	h.broadcast <- notification
}
