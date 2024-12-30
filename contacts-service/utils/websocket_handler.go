package utils

import (
	"contacts-service/models"
	"contacts-service/repository"
	"fmt"
	"log"
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
	Repo        repository.ContactsRepository
	Upgrader    websocket.Upgrader
}

func NewWebSocketHandler(repo repository.ContactsRepository, amqpChannel *amqp.Channel) *WebSocketHandler {
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
			Type         string `json:"type"`
			RequestID    string `json:"request_id"`
			UserID       string `json:"user_id"`
			Action       string `json:"action"` // accept or reject
			TargetUserID string `json:"target_user_id"`
		}
		err := conn.ReadJSON(&message)
		if err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
			h.Mutex.Lock()
			delete(h.Connections, userID)
			h.Mutex.Unlock()
			return
		}
		log.Print("Received message", message)
		switch message.Type {
		case "send_contact_request":
			h.HandleSendContactRequest(message.UserID, message.TargetUserID)
		case "accept_contact_request":
			h.HandleAcceptRejectContactRequest(message.RequestID, message.Action, userID, message.TargetUserID)
		case "reject_contact_request":
			h.HandleAcceptRejectContactRequest(message.RequestID, message.Action, userID, message.TargetUserID)
		}
	}
}

func (h *WebSocketHandler) HandleSendContactRequest(senderID, receiverID string) {
	// Save the contact request in the database

	contactRequest := models.ContactRequest{
		ID:         uuid.New(),
		SenderID:   uuid.MustParse(senderID),
		ReceiverID: uuid.MustParse(receiverID),
		Status:     "pending",
	}
	err := h.Repo.AddContactRequest(&contactRequest)
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
		if err := conn.WriteJSON(contactRequest); err != nil {
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

func (h *WebSocketHandler) HandleAcceptRejectContactRequest(requestID, action, userID, targetUserID string) {
	// Fetch the request
	log.Print("Accepting or rejecting contact request", requestID, action, userID, targetUserID)
	request, err := h.Repo.GetContactRequestByID(requestID)
	if err != nil {
		log.Printf("Failed to fetch contact request: %v", err)
		return
	}

	if action == "accept" {
		request.Status = "accepted"
		// Add the contact to the database
		err := h.Repo.AcceptOrReject(&models.Contact{
			UserID:    request.SenderID,
			ContactID: request.ReceiverID,
		})
		if err != nil {
			log.Printf("Failed to add contact: %v", err)
			return
		}
	} else if action == "reject" {
		request.Status = "rejected"
	} else {
		log.Printf("Invalid action: %s", action)
		return
	}

	// Update the request in the database
	err = h.Repo.UpdateContactRequest(request)
	if err != nil {
		log.Printf("Failed to update request: %v", err)
		return
	}

	// Broadcast the update to the target user if online
	h.Mutex.Lock()
	targetConn, online := h.Connections[targetUserID]
	h.Mutex.Unlock()

	if online {
		err := targetConn.WriteJSON(map[string]string{
			"type":   "contact_request_update",
			"action": action,
			"status": request.Status,
			"id":     request.ID.String(),
		})
		if err != nil {
			log.Printf("Failed to send update to target user: %v", err)
		}
	} else {
		log.Printf("Target user %s is offline. No real-time update sent.", targetUserID)
	}
}
