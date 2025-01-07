package tests

import (
	"auth-service/handlers"
	"auth-service/models"
	"auth-service/repository"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestSearchContacts(t *testing.T) {
	db := setupTestDB() // Reuse the shared setupTestDB function
	userRepo := repository.NewUserRepository(db)
	handler := &handlers.Handler{
		UserRepo: userRepo,
	}

	// Helper function to create test users
	createTestUser := func(username, email string) {
		userRepo.CreateUser(&models.User{
			ID:       uuid.New(),
			Username: username,
			Email:    email,
		})
	}

	// Create test users
	createTestUser("user1", "user1@example.com")
	createTestUser("user2", "user2@example.com")
	createTestUser("searchable", "searchable@example.com")

	t.Run("Successful Search with Results", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/search?q=search", nil)
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to the request context
		ctx := context.WithValue(req.Context(), "user_id", "user1")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.SearchContacts(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		// Decode response
		var contacts []models.User
		err := json.NewDecoder(rr.Body).Decode(&contacts)
		assert.NoError(t, err)

		// Assert that the correct contact is returned
		assert.Len(t, contacts, 1)
		assert.Equal(t, "searchable", contacts[0].Username)
		assert.Equal(t, "searchable@example.com", contacts[0].Email)
	})

	t.Run("Successful Search with No Results", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/search?q=nonexistent", nil)
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to the request context
		ctx := context.WithValue(req.Context(), "user_id", "user1")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.SearchContacts(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)

		// Decode response
		var contacts []models.User
		err := json.NewDecoder(rr.Body).Decode(&contacts)
		assert.NoError(t, err)

		// Assert that no results are returned
		assert.Len(t, contacts, 0)
	})

	t.Run("Search Query Missing", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/search", nil)
		req.Header.Set("Content-Type", "application/json")

		// Add user ID to the request context
		ctx := context.WithValue(req.Context(), "user_id", "user1")
		req = req.WithContext(ctx)

		rr := httptest.NewRecorder()
		handler.SearchContacts(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Equal(t, "Search query is required\n", rr.Body.String())
	})

	t.Run("Unauthorized Access (No User ID in Context)", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/search?q=search", nil)
		req.Header.Set("Content-Type", "application/json")

		// Do not add user ID to the context
		rr := httptest.NewRecorder()
		handler.SearchContacts(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Equal(t, "Unauthorized: user_id is required\n", rr.Body.String())
	})
}
