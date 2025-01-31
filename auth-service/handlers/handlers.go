// Handlers for user authentication, registration, and user search functionalities.
// Author: Nadir Hussain

package handlers

import (
	"auth-service/models"
	"auth-service/repository"
	"auth-service/utils"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Handler is a struct to manage user and session repository.
type Handler struct {
	UserRepo    repository.UserRepository
	SessionRepo repository.SessionRepository
}

// Validates user credentials and establishes a session.
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

	user, err := h.UserRepo.GetUserByUsernameOrEmail(credentials.Username, credentials.Username)
	if err != nil || user == nil { // Check for nil user
		http.Error(w, `{"message": "Invalid username or user does not exist"}`, http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		return
	}

	// Compare password sent by user, and one stored in database
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		http.Error(w, `{"message": "Invalid password"}`, http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
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
		w.Header().Set("Content-Type", "application/json")
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
		w.Header().Set("Content-Type", "application/json")
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

// Invalidates the user session and clears the cookie.
func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "No session found", http.StatusUnauthorized)
		return
	}

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

	log.Print("Email", user.Email)
	log.Print("Username", user.Username)
	// Check if a user with the same username or email already exists
	if existingUser, _ := h.UserRepo.GetUserByUsernameOrEmail(user.Username, user.Email); existingUser != nil {
		isUsernameConflict := existingUser.Username == user.Username
		isEmailConflict := existingUser.Email == user.Email

		if isUsernameConflict && isEmailConflict {
			http.Error(w, "User with this username and email already exists\n", http.StatusConflict)
		} else if isUsernameConflict {
			http.Error(w, "User with this username already exists\n", http.StatusConflict)
		} else if isEmailConflict {
			http.Error(w, "User with this email already exists\n", http.StatusConflict)
		}
		return
	}

	// Performing validation on input data
	if err := utils.ValidateRegistrationInput(user.Username, user.Email, user.Password); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		log.Printf("Error creating user: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User registered successfully"))
}

// Handles password reset requests
func (h *Handler) ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Username    string `json:"username"`
		NewPassword string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Performing validation on input data
	if err := utils.ValidatePassword(request.NewPassword); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	// Fetch the user by their username or email whatever is provided by user
	user, err := h.UserRepo.GetUserByUsernameOrEmail(request.Username, request.Username)
	if err != nil || user == nil { // Check for nil user
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

// SearchContacts searches and retrieves contacts based on the query string, excluding the user-itself who searches
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
