package middleware

import (
	"log"
	"messaging-service/utils"
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

		log.Printf("User authenticated: %s", response.UserID)
		// Optionally: Add user details to request context for downstream handlers
		next.ServeHTTP(w, r)
	})
}
