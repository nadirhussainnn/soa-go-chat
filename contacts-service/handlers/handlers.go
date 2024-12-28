package handlers

import (
	"contacts-service/repository"
	"contacts-service/utils"
	"encoding/json"
	"log"
	"net/http"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

type ContactsHandler struct {
	Repo             repository.ContactsRepository
	WebSocketHandler *utils.WebSocketHandler // Add WebSocketHandler
	AMQPChannel      *amqp.Channel           // Add AMQPChannel
}

func (h *ContactsHandler) GetContacts(w http.ResponseWriter, r *http.Request) {

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

	// Fetch contacts
	contacts, err := h.Repo.GetContactsByUserID(userID)
	if err != nil {
		log.Printf("Failed to fetch contacts: %v", err)
		http.Error(w, "Failed to fetch contacts", http.StatusInternalServerError)
		return
	}

	// Fetch contact requests
	contactRequests, err := h.Repo.GetContactRequestsByUserID(userID)
	if err != nil {
		log.Printf("Failed to fetch contact requests: %v", err)
		http.Error(w, "Failed to fetch contact requests", http.StatusInternalServerError)
		return
	}
	// Combine results
	response := map[string]interface{}{
		"contacts":        contacts,
		"contactRequests": contactRequests,
	}

	// Send the combined response as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	log.Println("Response sent successfully")
}
