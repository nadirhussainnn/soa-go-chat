package utils

import (
	"fmt"
	"log"
	"messaging-service/models"
	"messaging-service/repository"
	"net/http"

	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/streadway/amqp"
)

type WebSocketHandler struct {
	Connections map[string]*websocket.Conn
	Mutex       sync.Mutex
	AMQPChannel *amqp.Channel
	Repo        repository.MessagesRepository
	Upgrader    websocket.Upgrader
}

func NewWebSocketHandler(repo repository.MessagesRepository, amqpChannel *amqp.Channel) *WebSocketHandler {
	return &WebSocketHandler{
		Connections: make(map[string]*websocket.Conn),
		AMQPChannel: amqpChannel,
		Repo:        repo,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "Missing user_id in WebSocket request", http.StatusBadRequest)
		return
	}

	// Upgrade HTTP request to WebSocket
	conn, err := h.Upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		return
	}

	// Register the WebSocket connection
	h.Mutex.Lock()
	h.Connections[userID] = conn
	h.Mutex.Unlock()
	log.Printf("WebSocket connection established for user: %s", userID)

	// Listen for incoming messages
	for {
		var message struct {
			SenderID   string `json:"sender_id"`
			ReceiverID string `json:"receiver_id"`
			Content    string `json:"content"`
		}
		err := conn.ReadJSON(&message)
		if err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
			h.Mutex.Lock()
			delete(h.Connections, userID)
			h.Mutex.Unlock()
			return
		}

		h.HandleSendContactRequest(message.SenderID, message.ReceiverID, message.Content)
	}
}

func (h *WebSocketHandler) HandleSendContactRequest(senderID, receiverID, content string) {
	// Save the contact request in the database

	newMessage := models.Message{
		ID:         uuid.New(),
		SenderID:   uuid.MustParse(senderID),
		ReceiverID: uuid.MustParse(receiverID),
		Content:    content,
	}
	err := h.Repo.SendMessage(&newMessage)
	if err != nil {
		log.Printf("Failed to store contact request in DB: %v", err)
		return
	}

	// Check if the recipient is online
	h.Mutex.Lock()
	conn, online := h.Connections[receiverID]
	h.Mutex.Unlock()

	if online {
		// Send the request in real-time
		if err := conn.WriteJSON(newMessage); err != nil {
			log.Printf("Failed to send real-time contact request to user %s: %v", receiverID, err)
		} else {
			log.Printf("Real-time contact request sent to user %s", receiverID)
		}
	} else {
		// Notify via AMQP for later delivery
		err := h.AMQPChannel.Publish(
			"",
			NOTIFICATION_SERVICE,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        []byte(fmt.Sprintf(`{"type":"contact_request", "user_id":"%s", "target_user_id":"%s"}`, senderID, receiverID)),
			},
		)
		if err != nil {
			log.Printf("Failed to notify offline user %s via AMQP: %v", receiverID, err)
		}
		log.Printf("User %s is offline. Notification queued via AMQP.", receiverID)
	}
}
