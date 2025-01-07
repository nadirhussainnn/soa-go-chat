package handlers

import (
	"auth-service/models"
	"auth-service/repository"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	UserRepo    repository.UserRepository
	SessionRepo repository.SessionRepository
}

// LoginHandler handles user login
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	JWT_SECRET := []byte(os.Getenv("JWT_SECRET"))

	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Fetch the user
	user, err := h.UserRepo.GetUserByUsername(credentials.Username)
	if err != nil {
		http.Error(w, `{"message": "Invalid username or user does not exist"}`, http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json") // Ensure the response is JSON
		return

	}
	// Compare passwords
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		http.Error(w, `{"message": "Invalid password"}`, http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json") // Ensure the response is JSON
		return
	}

	// Generate JWT
	session_id := uuid.New()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"session_id": session_id,
	})
	tokenString, err := token.SignedString(JWT_SECRET)
	if err != nil {
		http.Error(w, `{"message": "Failed to generate token"}`, http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json") // Ensure the response is JSON
		return
	}
	// Create a session
	session := &models.Session{
		ID:     session_id,
		UserID: user.ID,
		Token:  tokenString,
	}
	if err := h.SessionRepo.CreateSession(session); err != nil {
		http.Error(w, `{"message": "Failed to create session"}`, http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json") // Ensure the response is JSON
		return
	}
	// Set the session cookie
	new_cookie := &http.Cookie{
		Name:    "session_token",
		Value:   tokenString,
		Expires: time.Now().Add(120 * time.Minute),
		// HttpOnly: true,
		Path: "/",
	}
	http.SetCookie(w, new_cookie)
	w.WriteHeader(http.StatusOK)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"user_id":       session.UserID.String(),
		"session_token": tokenString,
		"username":      user.Username,
		"email":         user.Email,
	})
}

func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "No session found", http.StatusUnauthorized)
		return
	}

	log.Print("Cookie: ", cookie.Value)
	// Delete the session
	if err := h.SessionRepo.DeleteSession(cookie.Value); err != nil {
		http.Error(w, "Failed to delete session", http.StatusInternalServerError)
		return
	}

	// Clear the cookie
	http.SetCookie(w, &http.Cookie{
		Name:   "session_token",
		Value:  "",
		Path:   "/",
		MaxAge: -1, // Expire immediately
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Logged out successfully"))
}

// RegisterHandler handles user registration
func (h *Handler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}
	user.Password = string(hashedPassword)

	// Save the user
	if err := h.UserRepo.CreateUser(&user); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			http.Error(w, "User with this username or email already exists", http.StatusConflict)
			return
		}

		log.Printf("Error creating user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User registered successfully"))
}

// ForgotPasswordHandler handles password reset requests
func (h *Handler) ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Username    string `json:"username"`
		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Fetch the user
	user, err := h.UserRepo.GetUserByUsername(request.Username)
	if err != nil {
		http.Error(w, "User with this username or does not exist", http.StatusConflict)
		return
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	// Update the user's password
	user.Password = string(hashedPassword)
	if err := h.UserRepo.UpdateUser(user); err != nil {
		http.Error(w, "Failed to update password", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Password updated successfully"))
}

func (h *Handler) SearchContacts(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query().Get("q") // Get the search query from the request

	if query == "" {
		http.Error(w, "Search query is required", http.StatusBadRequest)
		return
	}

	userID, ok := r.Context().Value("user_id").(string)

	if !ok || userID == "" {
		log.Print("User ID not found in context")
		http.Error(w, "Unauthorized: user_id is required", http.StatusUnauthorized)
		return
	}

	contacts, err := h.UserRepo.SearchUser(query, userID)
	if err != nil {
		http.Error(w, "Failed to search Users", http.StatusInternalServerError)
		return
	}

	// Send the matching contacts as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contacts)
}
