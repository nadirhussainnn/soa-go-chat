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
	query := r.URL.Query().Get("user_id")
	if query == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(query)
	if err != nil {
		http.Error(w, "Invalid user_id format", http.StatusBadRequest)
		return
	}

	// Fetch messages for the user
	messages, err := h.Repo.GetMessagesByUserID(userID)
	if err != nil {
		log.Printf("Failed to fetch messages: %v", err)
		http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
		return
	}

	// Collect all unique user IDs for batch fetching details
	userIDs := map[string]struct{}{}
	for _, msg := range messages {
		userIDs[msg.SenderID.String()] = struct{}{}
		userIDs[msg.ReceiverID.String()] = struct{}{}
	}

	// Convert map keys to a slice
	userIDList := make([]string, 0, len(userIDs))
	for id := range userIDs {
		userIDList = append(userIDList, id)
	}

	// Fetch user details
	newChannel, err := h.AMQPConn.Channel()
	if err != nil {
		log.Printf("Failed to create RabbitMQ channel: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer newChannel.Close()

	userDetailsMap, err := utils.GetUsersDetails(newChannel, userIDList)
	if err != nil {
		log.Printf("Failed to fetch user details: %v", err)
		http.Error(w, "Failed to fetch user details", http.StatusInternalServerError)
		return
	}

	// Attach user details to messages
	for i, msg := range messages {
		if details, exists := userDetailsMap[msg.SenderID.String()]; exists {
			messages[i].ContactDetails = details
		}
	}

	// Return messages in response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
