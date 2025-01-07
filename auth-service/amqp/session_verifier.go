// Handles session verification requests via RabbitMQ queues.
// Author: Nadir Hussain

package amqp

import (
	"auth-service/repository"
	"auth-service/utils"
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
)

// Handles session validation logic via RabbitMQ.
type SessionVerifier struct {
	SessionRepo repository.SessionRepository
}

// Represents the structure for session verification requests.
type SessionVerificationRequest struct {
	SessionID string `json:"session_id"`
}

// Represents the structure for session verification responses.
type SessionVerificationResponse struct {
	Valid  bool   `json:"valid"`
	UserID string `json:"user_id,omitempty"`
}

// Listens to session verification requests and publishes responses.
// Parameters:
// - ch: RabbitMQ channel for consuming and publishing messages.
func (sv *SessionVerifier) ListenForSessionVerification(ch *amqp.Channel) {
	q, err := ch.QueueDeclare(
		utils.AUTH_SESSION_VERIFICATION, // Name of the request queue
		false,                           // Durable
		false,                           // Delete when unused
		false,                           // Exclusive
		false,                           // No-wait
		nil,                             // Arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	// Declare the response queue
	_, err = ch.QueueDeclare(
		utils.AUTH_SESSION_RESPONSE, // Name of the response queue
		false,                       // Durable
		false,                       // Delete when unused
		false,                       // Exclusive
		false,                       // No-wait
		nil,                         // Arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare auth-session-response queue: %v", err)
	}

	msgs, err := ch.Consume(
		q.Name, // Queue name
		"",     // Consumer tag
		true,   // Auto-acknowledge
		false,  // Exclusive
		false,  // No-local
		false,  // No-wait
		nil,    // Args
	)
	if err != nil {
		log.Fatalf("Failed to register consumer: %v", err)
	}

	go func() {
		for d := range msgs {
			var request SessionVerificationRequest
			if err := json.Unmarshal(d.Body, &request); err != nil {
				log.Printf("Failed to unmarshal session verification request: %v", err)
				continue
			}

			response := sv.verifySession(request.SessionID)

			responseBytes, err := json.Marshal(response)
			if err != nil {
				log.Printf("Failed to marshal session verification response: %v", err)
				continue
			}

			// Publish response to the response queue
			err = ch.Publish(
				"",                          // Exchange
				utils.AUTH_SESSION_RESPONSE, // Response queue name
				false,                       // Mandatory
				false,                       // Immediate
				amqp.Publishing{ContentType: "application/json", Body: responseBytes},
			)
			if err != nil {
				log.Printf("Failed to publish session verification response: %v", err)
			}
		}
	}()
	log.Println("Listening for session verification requests...")
}

// Validates the session ID by checking its existence in the database.
// Parameters:
// - sessionID: The session ID to validate.
// Returns:
// - SessionVerificationResponse indicating whether the session is valid.
func (sv *SessionVerifier) verifySession(sessionID string) SessionVerificationResponse {
	session, err := sv.SessionRepo.GetSessionByID(sessionID)
	if err != nil || session == nil {
		return SessionVerificationResponse{Valid: false}
	}
	return SessionVerificationResponse{Valid: true, UserID: session.UserID.String()} // Convert UserID to string
}
