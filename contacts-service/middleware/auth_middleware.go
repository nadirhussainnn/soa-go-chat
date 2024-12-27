package middleware

import (
	"contacts-service/utils"
	"encoding/json"
	"log"
	"net/http"

	"github.com/streadway/amqp"
)

type AuthMiddleware struct {
	AMQPChannel *amqp.Channel
}

type DecodeJWTRequest struct {
	SessionToken string `json:"session_token"`
}

type DecodeJWTResponse struct {
	Valid    bool   `json:"valid"`
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Error    string `json:"error,omitempty"`
}

func (a *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract session_token from the cookie
		cookie, err := r.Cookie("session_token")
		if err != nil || cookie.Value == "" {
			log.Printf("No session token provided")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		sessionToken := cookie.Value
		// Send request to auth-service to decode JWT
		request := DecodeJWTRequest{SessionToken: sessionToken}
		requestBytes, _ := json.Marshal(request)

		err = a.AMQPChannel.Publish(
			"",                    // Exchange
			utils.AUTH_JWT_DECODE, // Routing key
			false,                 // Mandatory
			false,                 // Immediate
			amqp.Publishing{
				ContentType: "application/json",
				Body:        requestBytes,
			},
		)
		if err != nil {
			log.Printf("Failed to publish JWT decode request: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Consume the response
		msgs, err := a.AMQPChannel.Consume(
			utils.AUTH_JWT_DECODE_RESPONSE, // Queue
			"",                             // Consumer tag
			true,                           // Auto-acknowledge
			false,                          // Exclusive
			false,                          // No-local
			false,                          // No-wait
			nil,                            // Args
		)
		if err != nil {
			log.Printf("Failed to consume JWT decode response: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Wait for the response
		for d := range msgs {
			var response DecodeJWTResponse
			if err := json.Unmarshal(d.Body, &response); err != nil {
				log.Printf("Failed to unmarshal JWT decode response: %v", err)
				continue
			}

			if response.Valid {
				// Add user details to context
				log.Printf("User authenticated: %s", response.Username)
				next.ServeHTTP(w, r)
				return
			} else {
				log.Printf("Invalid session: %s", response.Error)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
	})
}
