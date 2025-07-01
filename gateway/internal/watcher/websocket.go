package watcher

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/lumbrjx/codek7/gateway/pkg/utils"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from localhost for development
		// In production, you should implement proper origin checking
		return true
	},
}

// HandleWebSocket handles WebSocket connections
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get user ID middleware context
	userID, ok := utils.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized: Missing user ID", http.StatusUnauthorized)
		return
	}

	// Upgrade HTTP connection to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create new client
	client := &Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan Notification, 256),
	}

	// Register client with hub
	h.register <- client

	// Handle ping/pong to keep connection alive
	go func() {
		defer func() {
			h.unregister <- client
		}()

		for {
			// Read message (ping/pong or close)
			_, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				break
			}
		}
	}()
}
