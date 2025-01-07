// Listens to Decode JWT token to get username and email from it, because auth-service is responsible for user details handling.
// Listens to provide details (username, email) for batch of user_ids received via RabbitMQ
// Author: Nadir Hussain

package amqp

import (
	"auth-service/models"
	"auth-service/repository"
	"auth-service/utils"
	"encoding/json"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/streadway/amqp"
)

// Handles JWT decoding with a secret key.
type JWTDecoder struct {
	Secret string
}

// Represents a request to decode a JWT token.
type DecodeJWTRequest struct {
	SessionToken string `json:"session_token"`
}

// Represents the response of a JWT decoding operation.
type JWTDecodeResponse struct {
	UserID   string `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
	Valid    bool   `json:"valid"`
	Error    string `json:"error,omitempty"`
}

// Represents a request to get details for user_ids received in array
type BatchDetailsRequest struct {
	UserIDs []string `json:"user_ids"`
}

// Represents the response by providing details to user_ids.
type BatchDetailsResponse struct {
	UserDetails map[string]*models.SenderDetails `json:"user_details"`
}

// ListenForJWTDecode listens to RabbitMQ queues for JWT decode requests.
// It declares queues, sets up consumers, and processes JWT decode requests.
// Parameters:
// - conn: RabbitMQ connection.
func (jd *JWTDecoder) ListenForJWTDecode(conn *amqp.Connection) {

	// Declaring queues using a dedicated channel
	ch, err := conn.Channel()
	log.Print("Message received ....", ch)
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// For each service, i declare a separate queue. because i encountered an issue, like when msg it was intended to be for contacts-service, then it was received to other services as well. Hence a separate queue for each service
	queues := []string{
		utils.AUTH_JWT_DECODE,
		utils.AUTH_JWT_DECODE_RESPONSE,
		utils.AUTH_JWT_DECODE_CONTACTS,
		utils.AUTH_JWT_DECODE_RESPONSE_CONTACTS,

		utils.AUTH_JWT_DECODE_MESSAGING,
		utils.AUTH_JWT_DECODE_RESPONSE_MESSAGING,
	}

	// Declaring all queueus
	for _, queue := range queues {
		declareQueue(ch, queue)
	}

	// Starting consumers with dedicated channels
	go jd.startConsumer(conn, utils.AUTH_JWT_DECODE, utils.AUTH_JWT_DECODE_RESPONSE)
	go jd.startConsumer(conn, utils.AUTH_JWT_DECODE_CONTACTS, utils.AUTH_JWT_DECODE_RESPONSE_CONTACTS)
	go jd.startConsumer(conn, utils.AUTH_JWT_DECODE_MESSAGING, utils.AUTH_JWT_DECODE_RESPONSE_MESSAGING)

	// Was getting issue in creating queues, asked GPT and it told to put a little pause so all queus are initialized properly
	time.Sleep(2 * time.Second)

	log.Println("Listening for JWT decode requests...")
}

// Declares a RabbitMQ queue.
// Parameters:
// - ch: RabbitMQ channel.
// - queueName: Name of the queue to declare.
func declareQueue(ch *amqp.Channel, queueName string) {
	_, err := ch.QueueDeclare(
		queueName, // Queue name
		false,     // Durable
		false,     // Delete when unused
		false,     // Exclusive
		false,     // No-wait
		nil,       // Arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare queue '%s': %v", queueName, err)
	}
}

// Starts a consumer for a request and response queue.
// Parameters:
// - conn: RabbitMQ connection.
// - requestQueue: Name of the request queue.
// - responseQueue: Name of the response queue.
func (jd *JWTDecoder) startConsumer(conn *amqp.Connection, requestQueue, responseQueue string) {
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel for queue '%s': %v", requestQueue, err)
	}

	msgs, err := ch.Consume(
		requestQueue, // Queue name
		"",           // Consumer tag
		true,         // Auto-acknowledge
		false,        // Exclusive
		false,        // No-local
		false,        // No-wait
		nil,          // Args
	)
	if err != nil {
		log.Fatalf("Failed to register consumer for queue '%s': %v", requestQueue, err)
	}

	log.Print("Messages", msgs)
	go func() {
		for d := range msgs {
			log.Printf("Received message on queue '%s': %s", requestQueue, string(d.Body))

			var request DecodeJWTRequest
			err := json.Unmarshal(d.Body, &request)
			if err != nil {
				log.Printf("Failed to unmarshal JWT decode request on queue '%s': %v", requestQueue, err)
				continue
			}

			// Decode the JWT
			response := jd.DecodeJWT(request.SessionToken)
			responseBytes, err := json.Marshal(response)
			if err != nil {
				log.Printf("Failed to marshal JWT decode response on queue '%s': %v", responseQueue, err)
				continue
			}

			// Publish response to the corresponding response queue
			err = ch.Publish(
				"",            // Exchange
				responseQueue, // Routing key (response queue)
				false,         // Mandatory
				false,         // Immediate
				amqp.Publishing{
					ContentType: "application/json",
					Body:        responseBytes,
				},
			)
			if err != nil {
				log.Printf("Failed to publish JWT decode response to queue '%s': %v", responseQueue, err)
			}
		}
	}()
}

// Decodes a JWT token and extracts claims.
// Parameters:
// - token: The JWT token string.
// Returns:
// - JWTDecodeResponse containing decoded user details or error information.
func (jd *JWTDecoder) DecodeJWT(token string) JWTDecodeResponse {

	tkn, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jd.Secret), nil
	})

	if err != nil || !tkn.Valid {
		return JWTDecodeResponse{
			Valid: false,
			Error: err.Error(),
		}
	}

	claims, ok := tkn.Claims.(jwt.MapClaims)
	if !ok {
		return JWTDecodeResponse{
			Valid: false,
			Error: "Invalid claims structure",
		}
	}

	return JWTDecodeResponse{
		Valid:    true,
		UserID:   claims["id"].(string),
		Username: claims["username"].(string),
		Email:    claims["email"].(string),
	}
}

// Listens for batch user details requests over RabbitMQ.
//
// Parameters:
// - channel: RabbitMQ channel for consuming messages.
// - userRepo: Repository to fetch user details from the database, and publish it back to the requesting queue
func ListenForBatchDetails(channel *amqp.Channel, userRepo repository.UserRepository) {

	_, err := channel.QueueDeclare(
		utils.AUTH_BATCH_DETAILS_REQUEST, // Queue name (request queue)
		false,                            // Durable
		false,                            // Delete when unused
		false,                            // Exclusive
		false,                            // No-wait
		nil,                              // Arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare request queue: %v", err)
	}
	log.Print("Received request for users details")
	// Declare the response queue
	_, err = channel.QueueDeclare(
		utils.AUTH_BATCH_DETAILS_RESPONSE, // Queue name (response queue)
		false,                             // Durable
		false,                             // Delete when unused
		false,                             // Exclusive
		false,                             // No-wait
		nil,                               // Arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare response queue: %v", err)
	}

	// Consume messages from the request queue
	msgs, err := channel.Consume(
		utils.AUTH_BATCH_DETAILS_REQUEST, // Queue name
		"",                               // Consumer tag
		true,                             // Auto-acknowledge
		false,                            // Exclusive
		false,                            // No-local
		false,                            // No-wait
		nil,                              // Args
	)
	if err != nil {
		log.Fatalf("Failed to consume from request queue: %v", err)
	}

	// Go routine to handle the user details retrieval from db, and to publish it to receivers queue
	go func() {
		log.Println("Started processing AUTH_BATCH_DETAILS_REQUEST messages")
		for d := range msgs {
			var request BatchDetailsRequest
			if err := json.Unmarshal(d.Body, &request); err != nil {
				log.Printf("Failed to unmarshal batch request: %v", err)
				continue
			}

			// Fetch user details
			userDetails := make(map[string]*models.SenderDetails)
			for _, userID := range request.UserIDs {
				user, err := userRepo.GetUserByID(userID)
				if err != nil {
					log.Printf("Failed to fetch user details for user_id %s: %v", userID, err)
					continue
				}
				userDetails[userID] = &models.SenderDetails{
					Username: user.Username,
					Email:    user.Email,
				}
			}

			// Prepare and publish the response
			response := BatchDetailsResponse{UserDetails: userDetails}
			responseBytes, err := json.Marshal(response)
			if err != nil {
				log.Printf("Failed to marshal response: %v", err)
				continue
			}
			log.Print("CorreltionID", d.CorrelationId)
			err = channel.Publish(
				"",                                // Exchange
				utils.AUTH_BATCH_DETAILS_RESPONSE, // Routing key (response queue)
				false,                             // Mandatory
				false,                             // Immediate
				amqp.Publishing{
					ContentType:   "application/json",
					Body:          responseBytes,
					CorrelationId: d.CorrelationId,
				},
			)
			if err != nil {
				log.Printf("Failed to publish response: %v", err)
			} else {
				log.Printf("Successfully published response to %s", utils.AUTH_BATCH_DETAILS_RESPONSE)
			}
		}
	}()
	log.Println("Listening for batch details requests...")
}
