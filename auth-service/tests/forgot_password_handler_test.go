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

func TestForgotPasswordHandler(t *testing.T) {
	db := setupTestDB() // Reuse the shared setupTestDB function
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	handler := &handlers.Handler{
		UserRepo:    userRepo,
		SessionRepo: sessionRepo,
	}

	// Helper function to create test users
	createTestUser := func(username, email, password string) {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		userRepo.CreateUser(&models.User{
			ID:       uuid.New(),
			Username: username,
			Email:    email,
			Password: string(hashedPassword),
		})
	}

	// Create a test user for password reset
	createTestUser("forgotuser", "forgotuser@example.com", "OldPassword@123")

	t.Run("Successful Password Reset", func(t *testing.T) {
		payload := map[string]string{
			"username":     "forgotuser",
			"new_password": "NewPassword@123",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.ForgotPasswordHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "Password updated successfully", rr.Body.String())
	})

	t.Run("Invalid Username", func(t *testing.T) {
		payload := map[string]string{
			"username":     "nonexistentuser", // Username does not exist
			"new_password": "NewPassword@123",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.ForgotPasswordHandler(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code)
		assert.Equal(t, "User with this username or does not exist\n", rr.Body.String())
	})

	t.Run("Short Password", func(t *testing.T) {
		payload := map[string]string{
			"username":     "forgotuser", // Valid user
			"new_password": "123",
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.ForgotPasswordHandler(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code)
		assert.Equal(t, "Password must be atleast 6 characters\n", rr.Body.String())
	})

	t.Run("No Special Character in Password", func(t *testing.T) {
		payload := map[string]string{
			"username":     "forgotuser",
			"new_password": "Password123", // No special character
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/forgot-password", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.ForgotPasswordHandler(rr, req)

		assert.Equal(t, http.StatusConflict, rr.Code)
		assert.Equal(t, "Password must include at least one special character\n", rr.Body.String())
	})
}
