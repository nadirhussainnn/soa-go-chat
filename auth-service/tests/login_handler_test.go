package tests

import (
	"auth-service/handlers"
	"auth-service/models"
	"auth-service/repository"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestLoginHandler(t *testing.T) {
	db := setupTestDB()
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	handler := &handlers.Handler{
		UserRepo:    userRepo,
		SessionRepo: sessionRepo,
	}

	// Helper function to create test user
	createTestUser := func(username, email, password string) {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		userRepo.CreateUser(&models.User{
			ID:       uuid.New(),
			Username: username,
			Email:    email,
			Password: string(hashedPassword),
		})
	}

	// Create a valid test user for login tests
	createTestUser("testuser", "testuser@example.com", "Secure@123")

	t.Run("Successful Login", func(t *testing.T) {
		credentials := map[string]string{
			"username": "testuser", // or "testuser@example.com"
			"password": "Secure@123",
		}
		body, _ := json.Marshal(credentials)
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.LoginHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		// Assert the response includes the session token and user details
		var response map[string]string
		err := json.NewDecoder(rr.Body).Decode(&response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response["session_token"])
		assert.Equal(t, "testuser", response["username"])
		assert.Equal(t, "testuser@example.com", response["email"])
	})

	t.Run("Incorrect Username or Email", func(t *testing.T) {
		credentials := map[string]string{
			"username": "nonexistentuser", // Username doesn't exist
			"password": "Secure@123",
		}
		body, _ := json.Marshal(credentials)
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.LoginHandler(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.JSONEq(t, `{"message": "Invalid username or user does not exist"}`, rr.Body.String())
	})

	t.Run("Incorrect Password", func(t *testing.T) {
		credentials := map[string]string{
			"username": "testuser",      // Valid username
			"password": "WrongPassword", // Incorrect password
		}
		body, _ := json.Marshal(credentials)
		req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.LoginHandler(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.JSONEq(t, `{"message": "Invalid password"}`, rr.Body.String())
	})
}
