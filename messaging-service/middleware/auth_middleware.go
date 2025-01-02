package middleware

import (
	"context"
	"log"
	"messaging-service/utils"
	"net/http"

	"github.com/streadway/amqp"
)

type AuthMiddleware struct {
	AMQPConn *amqp.Connection // Use the RabbitMQ connection, not a shared channel
}

// RequireAuth is the middleware for authenticating requests
func (a *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a new RabbitMQ channel for this request
		ch, err := a.AMQPConn.Channel()
		if err != nil {
			log.Printf("Failed to create RabbitMQ channel: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer ch.Close()

		// Extract session_token from the cookie
		cookie, err := r.Cookie("session_token")
		if err != nil || cookie.Value == "" {
			log.Printf("No session token provided")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Decode the JWT using the helper function
		response, err := utils.DecodeJWT(ch, cookie.Value)
		log.Print("Decode Response", response)

		if err != nil || !response.Valid {
			log.Printf("Invalid session: %s", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		log.Print("User ID", response.UserID)
		// Extract user_id from session token and add it to request context
		ctx := context.WithValue(r.Context(), "user_id", response.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
