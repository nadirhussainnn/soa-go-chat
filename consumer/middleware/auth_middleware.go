// Middleware for handling JWT-based authentication and request authorization.
// Author: Nadir Hussain

package middleware

import (
	"consumer/utils"
	"context"
	"log"
	"net/http"

	"github.com/streadway/amqp"
)

type AuthMiddleware struct {
	AMQPConn *amqp.Connection
}

// RequireAuth is the middleware for authenticating requests and protecting the pages/APIs
func (a *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create a new RabbitMQ channel for this request
		ch, err := a.AMQPConn.Channel()
		if err != nil {
			log.Printf("Failed to create RabbitMQ channel: %v", err)
			utils.RenderErrorPage(w, "Internal server error")
			return
		}
		defer ch.Close()

		// Extract session_token from the cookie
		cookie, err := r.Cookie("session_token")
		if err != nil || cookie.Value == "" {
			log.Printf("No session token provided")
			utils.RenderErrorPage(w, "Unauthorized: No session token provided")

			return
		}

		// Decode the JWT using the helper function
		response, err := utils.DecodeJWT(ch, cookie.Value)
		log.Print("Decode Response", response)

		if err != nil || !response.Valid {
			log.Printf("Invalid session: %s", err)
			utils.RenderErrorPage(w, "Unauthorized: Invalid session token")
			return
		}
		log.Print("Passing [user] context to route", response.UserID, response.Username, response.Email)
		// Extract user_id from session token and add it to request context
		ctx := context.WithValue(r.Context(), "user_id", response.UserID)
		ctx = context.WithValue(ctx, "username", response.Username)
		ctx = context.WithValue(ctx, "email", response.Email)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
