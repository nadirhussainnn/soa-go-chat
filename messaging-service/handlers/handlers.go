package handlers

import (
	"encoding/json"
	"log"
	"messaging-service/repository"
	"messaging-service/utils"
	"net/http"
	"os"
	"path/filepath"

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

func (h *MessageHandler) ServeFile(w http.ResponseWriter, r *http.Request) {
	// Get the message ID from the query parameter
	messageIDStr := r.URL.Query().Get("message_id")
	if messageIDStr == "" {
		http.Error(w, "message_id is required", http.StatusBadRequest)
		return
	}

	// Parse the message ID as a UUID
	messageID, err := uuid.Parse(messageIDStr)
	if err != nil {
		http.Error(w, "Invalid message_id format", http.StatusBadRequest)
		return
	}

	// Fetch the message by ID
	message, err := h.Repo.GetMessageByID(messageID)
	if err != nil {
		http.Error(w, "Message not found", http.StatusNotFound)
		return
	}

	// Ensure the message has a file associated with it
	if message.MessageType != "file" || message.FilePath == "" {
		http.Error(w, "No file associated with this message", http.StatusBadRequest)
		return
	}

	// Construct the file path
	filePath := filepath.Join("./uploads", message.FilePath)

	// Check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.Error(w, "File not found on the server", http.StatusNotFound)
		return
	}

	// Serve the file with the original filename
	w.Header().Set("Content-Disposition", "attachment; filename=\""+message.FileName+"\"")
	w.Header().Set("Content-Type", message.FileMimeType)
	http.ServeFile(w, r, filePath)
}
