package api

import (
	"net/http"
)

// WebSocketHandler handles WebSocket connections for notifications
func (a *API) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	if a.Hub != nil {
		a.Hub.HandleWebSocket(w, r)
	} else {
		http.Error(w, "WebSocket hub not initialized", http.StatusInternalServerError)
	}
}

func (a *API) ErHandler(w http.ResponseWriter, r *http.Request) {
	if a.Hub != nil {
		a.Hub.HandleWebSocket(w, r)
	} else {
		http.Error(w, "WebSocket hub not initialized", http.StatusInternalServerError)
	}
}
