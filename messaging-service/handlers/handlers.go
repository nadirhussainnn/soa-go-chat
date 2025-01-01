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
	Repo             repository.MessagesRepository
	WebSocketHandler *utils.WebSocketHandler // Add WebSocketHandler
	AMQPChannel      *amqp.Channel           // Add AMQPChannel
}

func (h *MessageHandler) GetMessages(w http.ResponseWriter, r *http.Request) {

	log.Println("Getting messaging", r.URL)
	query := r.URL.Query().Get("user_id") // Get the search query from the request

	if query == "" {
		http.Error(w, "Search query is required", http.StatusBadRequest)
		return
	}

	messaging, err := h.Repo.GetMessagesByUserID(uuid.MustParse(query))
	if err != nil {
		http.Error(w, "Failed to search Users", http.StatusInternalServerError)
		return
	}

	// Send the matching messaging as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messaging)
}
