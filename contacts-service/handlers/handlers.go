package handlers

import (
	"contacts-service/models"
	"contacts-service/repository"
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/google/uuid"
)

type ContactsHandler struct {
	Repo repository.ContactsRepository
}

func (h *ContactsHandler) AcceptOrReject(w http.ResponseWriter, r *http.Request) {
	var rawData struct {
		UserID    string `json:"user_id"`
		ContactID string `json:"contact_id"`
	}

	// Read and log the request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	log.Printf("Raw Request Body: %s", string(bodyBytes))

	// Decode the body into a temporary struct
	if err := json.Unmarshal(bodyBytes, &rawData); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Parse UUIDs from the raw data
	userID, err := uuid.Parse(rawData.UserID)
	if err != nil {
		http.Error(w, "Invalid user_id format", http.StatusBadRequest)
		return
	}
	contactID, err := uuid.Parse(rawData.ContactID)
	if err != nil {
		http.Error(w, "Invalid contact_id format", http.StatusBadRequest)
		return
	}

	contact := models.Contact{
		ID:        uuid.New(),
		UserID:    userID,
		ContactID: contactID,
	}

	log.Printf("Decoded Contact: %+v", contact)

	// Save the contact to the repository
	if err := h.Repo.AcceptOrReject(&contact); err != nil {
		http.Error(w, "Failed to add contact", http.StatusInternalServerError)
		return
	}

	// Respond with the created contact
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(contact)

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

// HandleContactRequest processes contact requests (accept/reject)
func (h *ContactsHandler) SendContactRequest(w http.ResponseWriter, r *http.Request) {
	var req models.ContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if err := h.Repo.AddContactRequest(&req); err != nil {
		http.Error(w, "Failed to process contact request", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Contact request updated successfully"))
}
