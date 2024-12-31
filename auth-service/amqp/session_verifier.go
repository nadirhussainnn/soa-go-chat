package amqp

import (
	"auth-service/repository"
	"auth-service/utils"
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
)

type SessionVerifier struct {
	SessionRepo repository.SessionRepository
}

type SessionVerificationRequest struct {
	SessionID string `json:"session_id"`
}

type SessionVerificationResponse struct {
	Valid  bool   `json:"valid"`
	UserID string `json:"user_id,omitempty"`
}

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
				"",                      // Exchange
				"auth-session-response", // Response queue name
				false,                   // Mandatory
				false,                   // Immediate
				amqp.Publishing{ContentType: "application/json", Body: responseBytes},
			)
			if err != nil {
				log.Printf("Failed to publish session verification response: %v", err)
			}
		}
	}()
	log.Println("Listening for session verification requests...")
}

func (sv *SessionVerifier) verifySession(sessionID string) SessionVerificationResponse {
	session, err := sv.SessionRepo.GetSessionByID(sessionID)
	if err != nil || session == nil {
		return SessionVerificationResponse{Valid: false}
	}
	return SessionVerificationResponse{Valid: true, UserID: session.UserID.String()} // Convert UserID to string
}

// InitRabbitMQ sets up the RabbitMQ connection and channel
func InitRabbitMQ(amqpURL string) (*amqp.Connection, *amqp.Channel) {
	// Connect to RabbitMQ
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	// Open a channel
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	return conn, ch
}
