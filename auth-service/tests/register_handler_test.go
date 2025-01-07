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

	"github.com/stretchr/testify/assert"
)

func TestRegisterHandler(t *testing.T) {
	db := setupTestDB()
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	handler := &handlers.Handler{
		UserRepo:    userRepo,
		SessionRepo: sessionRepo,
	}

	t.Run("Successful Register", func(t *testing.T) {
		testUser := models.User{
			Username: "newuser",
			Email:    "newuser@example.com",
			Password: "Secure@123",
		}
		body, _ := json.Marshal(testUser)
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.RegisterHandler(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		assert.Equal(t, "User registered successfully", rr.Body.String())
	})

	t.Run("Failed Register - Short Password", func(t *testing.T) {
		testUser := models.User{
			Username: "shortpassworduser",
			Email:    "shortpassworduser@example.com",
			Password: "123",
		}
		body, _ := json.Marshal(testUser)
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.RegisterHandler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Equal(t, "Password must be atleast 6 characters\n", rr.Body.String()) // Correct message and newline
	})

	t.Run("Failed Register - All Lowercase Password", func(t *testing.T) {
		testUser := models.User{
			Username: "lowercasepassword",
			Email:    "lowercase@example.com",
			Password: "password",
		}
		body, _ := json.Marshal(testUser)
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.RegisterHandler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Equal(t, "Password must include at least one uppercase letter\n", rr.Body.String()) // Include newline
	})

	t.Run("Failed Register - No Special Character", func(t *testing.T) {
		testUser := models.User{
			Username: "nospecialchar",
			Email:    "nospecial@example.com",
			Password: "Password123",
		}
		body, _ := json.Marshal(testUser)
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.RegisterHandler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Equal(t, "Password must include at least one special character\n", rr.Body.String()) // Include newline
	})

	t.Run("Failed Register - No Number", func(t *testing.T) {
		testUser := models.User{
			Username: "nonumber",
			Email:    "nonumber@example.com",
			Password: "Password@",
		}
		body, _ := json.Marshal(testUser)
		req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()
		handler.RegisterHandler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Equal(t, "Password must include at least one number\n", rr.Body.String()) // Include newline
	})

}
