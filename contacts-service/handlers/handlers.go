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
	WebSocketHandler *utils.WebSocketHandler
	AMQPConn         *amqp.Connection // Store RabbitMQ connection
}

func (h *ContactsHandler) GetContacts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("user_id")
	log.Print("Query: ", query)
	if query == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(query)
	if err != nil {
		http.Error(w, "Invalid user_id format", http.StatusBadRequest)
		return
	}

	// Create a new RabbitMQ channel for this request
	newChannel, err := h.AMQPConn.Channel()
	if err != nil {
		log.Printf("Failed to create RabbitMQ channel: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer newChannel.Close()

	log.Print("fetching contacts of UserID", userID)
	// Fetch contacts for the given user
	contacts, err := h.Repo.GetContactsByUserID(userID)
	if err != nil {
		log.Printf("Failed to fetch contacts: %v", err)
		http.Error(w, "Failed to fetch contacts", http.StatusInternalServerError)
		return
	}
	log.Print("fetched contacts", contacts)
	// Collect all user IDs for batch fetching details
	userIDs := []string{}
	for _, contact := range contacts {
		userIDs = append(userIDs, contact.ContactID.String())
	}

	log.Print("Sending UserIDs for details: ", userIDs)
	// Fetch user details in batch
	userDetailsMap, err := utils.GetUsersDetails(newChannel, userIDs)
	if err != nil {
		log.Printf("Failed to fetch user details: %v", err)
		http.Error(w, "Failed to fetch user details", http.StatusInternalServerError)
		return
	}

	log.Print("Maps: ", userDetailsMap)

	// Map user details to contacts
	for i, contact := range contacts {
		if details, exists := userDetailsMap[contact.ContactID.String()]; exists {
			contacts[i].ContactDetails = details
			contacts[i].ContactDetails.UserID = contact.ContactID.String()
		}
	}

	// Logging details
	for userID, details := range userDetailsMap {
		log.Printf("UserID: %s, Details: %+v", userID, details)
	}

	// Wrap the response in a "contacts" field
	response := map[string]interface{}{
		"contacts": contacts,
	}

	// Send the response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (h *ContactsHandler) FetchPendingRequests(w http.ResponseWriter, r *http.Request) {
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

	// Create a new RabbitMQ channel for this request
	newChannel, err := h.AMQPConn.Channel()
	if err != nil {
		log.Printf("Failed to create RabbitMQ channel: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer newChannel.Close()

	// Fetch pending contact requests
	requests, err := h.Repo.GetContactRequestsByUserID(userID)
	if err != nil {
		log.Printf("Failed to fetch contact requests: %v", err)
		http.Error(w, "Failed to fetch contact requests", http.StatusInternalServerError)
		return
	}

	// Collect all user IDs for batch fetching details
	userIDs := []string{}
	for _, req := range requests {
		userIDs = append(userIDs, req.SenderID.String(), req.ReceiverID.String())
	}

	// Fetch user details in batch
	userDetailsMap, err := utils.GetUsersDetails(newChannel, userIDs)
	if err != nil {
		log.Printf("Failed to fetch user details: %v", err)
		http.Error(w, "Failed to fetch user details", http.StatusInternalServerError)
		return
	}

	for userID, details := range userDetailsMap {
		log.Printf("UserID: %s, Details: %+v", userID, details)
	}

	// Map user details to requests
	for i, req := range requests {
		if details, exists := userDetailsMap[req.SenderID.String()]; exists {
			requests[i].SenderDetails = details
			requests[i].SenderDetails.UserID = req.SenderID.String() // Add UserID explicitly

		}
		if details, exists := userDetailsMap[req.ReceiverID.String()]; exists {
			requests[i].TargetUserDetails = details
			requests[i].TargetUserDetails.UserID = userID.String()
		}
		requests[i].CreatedAtFormatted = req.CreatedAt.Format("2 Jan, 2006")
	}
	log.Print("Received request for requests", requests)
	// Send response with pending requests only
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(requests); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
