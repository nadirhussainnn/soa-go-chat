package utils

import (
	"log"
	"net/http"

	"sync"

	"github.com/gorilla/websocket"
)

type WebSocketHandler struct {
	Connections map[string]*websocket.Conn // user_id -> WebSocket connection
	Mutex       sync.Mutex
}

func NewWebSocketHandler() *WebSocketHandler {
	return &WebSocketHandler{
		Connections: make(map[string]*websocket.Conn),
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity; restrict as needed
	},
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket: %v", err)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		log.Printf("Missing user_id in WebSocket request")
		conn.Close()
		return
	}

	h.Mutex.Lock()
	h.Connections[userID] = conn
	h.Mutex.Unlock()

	log.Printf("WebSocket connection established for userID: %s", userID)

	defer func() {
		h.Mutex.Lock()
		delete(h.Connections, userID)
		h.Mutex.Unlock()
		conn.Close()
		log.Printf("WebSocket connection closed for userID: %s", userID)
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error for userID %s: %v", userID, err)
			break
		}
	}
}
