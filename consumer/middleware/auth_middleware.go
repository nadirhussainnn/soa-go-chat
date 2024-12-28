package middleware

import (
	"consumer/utils"
	"context"
	"log"
	"net/http"

	"github.com/streadway/amqp"
)

type AuthMiddleware struct {
	AMQPChannel *amqp.Channel
}

// RequireAuth is the middleware for authenticating requests
func (a *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract session_token from the cookie
		cookie, err := r.Cookie("session_token")
		if err != nil || cookie.Value == "" {
			log.Printf("No session token provided")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Decode the JWT using the helper function
		response, err := utils.DecodeJWT(a.AMQPChannel, cookie.Value)
		if err != nil || !response.Valid {
			log.Printf("Invalid session: %s", err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// next.ServeHTTP(w, r)

		// Extract user_id from session token and add it to request context
		ctx := context.WithValue(r.Context(), "user_id", response.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))

	})
}
