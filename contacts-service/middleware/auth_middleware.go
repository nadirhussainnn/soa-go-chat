package middleware

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/streadway/amqp"
)

type AuthMiddleware struct {
	AMQPChannel *amqp.Channel
}

func (a *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		sessionToken := r.Header.Get("Authorization")
		if sessionToken == "" {
			log.Printf("No session token provided")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Remove Bearer prefix
		if len(sessionToken) > 7 && sessionToken[:7] == "Bearer " {
			sessionToken = sessionToken[7:]
		}

		// Publish session verification request
		err := a.AMQPChannel.Publish(
			"",                          // Exchange
			"auth-session-verification", // Routing key
			false,                       // Mandatory
			false,                       // Immediate
			amqp.Publishing{
				ContentType: "application/json",
				Body:        []byte(`{"session_id":"` + sessionToken + `"}`),
			},
		)
		if err != nil {
			log.Printf("Failed to publish session verification: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Consume session verification response
		msgs, err := a.AMQPChannel.Consume(
			"auth-session-response", // Queue
			"",                      // Consumer
			true,                    // Auto-acknowledge
			false,                   // Exclusive
			false,                   // No-local
			false,                   // No-wait
			nil,                     // Args
		)
		if err != nil {
			log.Printf("Failed to consume session response: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Wait for response
		for d := range msgs {
			var response map[string]interface{}
			json.Unmarshal(d.Body, &response)

			if valid, ok := response["valid"].(bool); ok && valid {
				// Authorized
				log.Print("Authorized", response)
				next.ServeHTTP(w, r)
				return
			}
			break
		}
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// package middleware

// import (
// 	"encoding/json"
// 	"log"
// 	"net/http"
// 	"sync"
// 	"time"

// 	"github.com/streadway/amqp"
// )

// type AuthMiddleware struct {
// 	AMQPChannel *amqp.Channel
// }

// type CacheItem struct {
// 	UserID    string
// 	Timestamp time.Time
// }

// var (
// 	sessionCache = make(map[string]CacheItem)
// 	cacheMutex   sync.Mutex
// 	cacheTTL     = 5 * time.Minute // Cache TTL is 5 minutes
// )

// func (a *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		sessionToken := r.Header.Get("Authorization")
// 		if sessionToken == "" {
// 			log.Printf("No session token provided")
// 			http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 			return
// 		}

// 		// Remove "Bearer " prefix
// 		if len(sessionToken) > 7 && sessionToken[:7] == "Bearer " {
// 			sessionToken = sessionToken[7:]
// 		}

// 		// Check the cache
// 		cacheMutex.Lock()
// 		if cachedItem, found := sessionCache[sessionToken]; found {
// 			if time.Since(cachedItem.Timestamp) < cacheTTL {
// 				// Cache hit and valid
// 				cacheMutex.Unlock()
// 				log.Printf("Cache hit for sessionToken: %s", sessionToken)
// 				r = r.WithContext(r.Context())
// 				next.ServeHTTP(w, r)
// 				return
// 			}
// 			// Cache expired, delete the entry
// 			delete(sessionCache, sessionToken)
// 		}
// 		cacheMutex.Unlock()

// 		// Publish session verification request
// 		err := a.AMQPChannel.Publish(
// 			"",                          // Exchange
// 			"auth-session-verification", // Routing key
// 			false,                       // Mandatory
// 			false,                       // Immediate
// 			amqp.Publishing{
// 				ContentType: "application/json",
// 				Body:        []byte(`{"session_id":"` + sessionToken + `"}`),
// 			},
// 		)
// 		if err != nil {
// 			log.Printf("Failed to publish session verification: %v", err)
// 			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 			return
// 		}

// 		// Consume session verification response
// 		msgs, err := a.AMQPChannel.Consume(
// 			"auth-session-response", // Queue
// 			"",                      // Consumer
// 			true,                    // Auto-acknowledge
// 			false,                   // Exclusive
// 			false,                   // No-local
// 			false,                   // No-wait
// 			nil,                     // Args
// 		)
// 		if err != nil {
// 			log.Printf("Failed to consume session response: %v", err)
// 			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 			return
// 		}

// 		// Wait for response
// 		for d := range msgs {
// 			var response map[string]interface{}
// 			err := json.Unmarshal(d.Body, &response)
// 			if err != nil {
// 				log.Printf("Failed to unmarshal session verification response: %v", err)
// 				break
// 			}

// 			if valid, ok := response["valid"].(bool); ok && valid {
// 				userID, _ := response["user_id"].(string)

// 				// Cache the session token
// 				cacheMutex.Lock()
// 				sessionCache[sessionToken] = CacheItem{
// 					UserID:    userID,
// 					Timestamp: time.Now(),
// 				}
// 				cacheMutex.Unlock()

// 				// Add userID to the request context for downstream handlers
// 				r = r.WithContext(r.Context())
// 				next.ServeHTTP(w, r)
// 				return
// 			}
// 			break
// 		}

// 		// Unauthorized
// 		http.Error(w, "Unauthorized", http.StatusUnauthorized)
// 	})
// }
