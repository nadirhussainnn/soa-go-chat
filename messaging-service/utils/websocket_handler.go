package utils

import (
	"fmt"
	"log"
	"messaging-service/models"
	"messaging-service/repository"
	"net/http"
	"time"

	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/streadway/amqp"
)

type WebSocketHandler struct {
	Connections map[string]*websocket.Conn
	Mutex       sync.Mutex
	AMQPChannel *amqp.Channel
	Repo        repository.MessageRepository
	Upgrader    websocket.Upgrader
}

func NewWebSocketHandler(repo repository.MessageRepository, amqpChannel *amqp.Channel) *WebSocketHandler {
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
			Type       string `json:"type"`
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

		log.Print("Message received ", message.Content, message.ReceiverID, message.SenderID, message.Type)
		// Handle new message
		if message.Type == "send_message" {
			h.HandleNewMessage(message.SenderID, message.ReceiverID, message.Content)
		}
	}
}

func (h *WebSocketHandler) HandleNewMessage(senderID, receiverID, content string) {
	// Create and save the message in the database
	log.Print("Sending message --", content)
	message := models.Message{
		ID:         uuid.New(),
		SenderID:   uuid.MustParse(senderID),
		ReceiverID: uuid.MustParse(receiverID),
		Content:    content,
		CreatedAt:  time.Now(),
	}
	err := h.Repo.CreateNewMessage(&message)
	if err != nil {
		log.Printf("Failed to save message: %v", err)
		return
	}

	// Notify the sender
	h.Mutex.Lock()
	senderConn, senderOnline := h.Connections[senderID]
	h.Mutex.Unlock()

	if senderOnline {
		if err := senderConn.WriteJSON(map[string]interface{}{
			"type":    MESSAGE_SENT_ACK,
			"message": message,
		}); err != nil {
			log.Printf("Failed to notify sender %s: %v", senderID, err)
		}
	}

	// Notify the receiver
	h.Mutex.Lock()
	receiverConn, receiverOnline := h.Connections[receiverID]
	h.Mutex.Unlock()

	if receiverOnline {
		if err := receiverConn.WriteJSON(map[string]interface{}{
			"type":    NEW_MESSAGE_RECEIVED,
			"message": message,
		}); err != nil {
			log.Printf("Failed to notify receiver %s: %v", receiverID, err)
		}
	} else {
		// Notify offline user via RabbitMQ
		err := h.AMQPChannel.Publish(
			"",
			NOTIFICATION_SERVICE,
			false,
			false,
			amqp.Publishing{
				ContentType: "application/json",
				Body:        []byte(fmt.Sprintf(`{"type":"new_message", "user_id":"%s", "content":"%s"}`, receiverID, content)),
			},
		)
		if err != nil {
			log.Printf("Failed to send message notification for offline user %s: %v", receiverID, err)
		}
	}
}
