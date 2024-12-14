package handlers

import (
	"auth-service/repository"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type ContactHandler struct {
	ContactRepo *repository.ContactRepository
}

func (h *ContactHandler) FetchAvailableContacts(w http.ResponseWriter, r *http.Request) {
	userID := 1 // Simulate logged-in user ID
	contacts, err := h.ContactRepo.GetAvailableUsers(uint(userID))
	if err != nil {
		http.Error(w, "Failed to fetch contacts", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(contacts)
}

func (h *ContactHandler) FetchUserContacts(w http.ResponseWriter, r *http.Request) {
	userID := 1 // Simulate logged-in user ID
	contacts, err := h.ContactRepo.GetUserContacts(uint(userID))
	if err != nil {
		http.Error(w, "Failed to fetch user contacts", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(contacts)
}

func (h *ContactHandler) SearchContacts(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	userID := 1 // Simulate logged-in user ID
	contacts, err := h.ContactRepo.SearchUsers(query, uint(userID))
	if err != nil {
		http.Error(w, "Failed to search contacts", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(contacts)
}

func (h *ContactHandler) SendContactRequest(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	receiverID, _ := strconv.Atoi(params["id"])
	senderID := 1 // Simulate logged-in user ID
	err := h.ContactRepo.SendContactRequest(uint(senderID), uint(receiverID))
	if err != nil {
		http.Error(w, "Failed to send contact request", http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Request sent"))
}

func (h *ContactHandler) RemoveContact(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	contactID, _ := strconv.Atoi(params["id"])
	userID := 1 // Simulate logged-in user ID
	err := h.ContactRepo.RemoveContact(uint(userID), uint(contactID))
	if err != nil {
		http.Error(w, "Failed to remove contact", http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Contact removed"))
}
