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

	log.Println("Getting contacts", r.URL)
	query := r.URL.Query().Get("user_id") // Get the search query from the request

	if query == "" {
		http.Error(w, "Search query is required", http.StatusBadRequest)
		return
	}

	contacts, err := h.Repo.GetContactsByUserID(uuid.MustParse(query))
	if err != nil {
		http.Error(w, "Failed to search Users", http.StatusInternalServerError)
		return
	}

	// Send the matching contacts as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contacts)
}
