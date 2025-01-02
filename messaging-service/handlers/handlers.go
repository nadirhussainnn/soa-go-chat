package handlers

import (
	"encoding/json"
	"log"
	"messaging-service/repository"
	"messaging-service/utils"
	"net/http"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

type MessageHandler struct {
	Repo             repository.MessageRepository
	WebSocketHandler *utils.WebSocketHandler
	AMQPConn         *amqp.Connection // Store RabbitMQ connection
}

func (h *MessageHandler) FetchMessages(w http.ResponseWriter, r *http.Request) {
	userIdStr := r.URL.Query().Get("user_id")
	contactIdStr := r.URL.Query().Get("contact_id")

	if userIdStr == "" || contactIdStr == "" {
		http.Error(w, "user_id and contact_id are required", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(userIdStr)
	if err != nil {
		http.Error(w, "Invalid user_id format", http.StatusBadRequest)
		return
	}

	contactID, err := uuid.Parse(contactIdStr)
	if err != nil {
		http.Error(w, "Invalid contact_id format", http.StatusBadRequest)
		return
	}

	// Fetch messages for the user
	messages, err := h.Repo.GetMessagesByUserID(userID, contactID)
	if err != nil {
		log.Printf("Failed to fetch messages: %v", err)
		http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
		return
	}

	log.Print(messages[0])
	// Return messages in response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
