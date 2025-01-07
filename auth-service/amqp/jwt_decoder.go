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

type JWTDecoder struct {
	Secret string
}

type DecodeJWTRequest struct {
	SessionToken string `json:"session_token"`
}

type JWTDecodeResponse struct {
	UserID   string `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
	Valid    bool   `json:"valid"`
	Error    string `json:"error,omitempty"`
}

type BatchDetailsRequest struct {
	UserIDs []string `json:"user_ids"`
}

type BatchDetailsResponse struct {
	UserDetails map[string]*models.SenderDetails `json:"user_details"`
}

func (jd *JWTDecoder) ListenForJWTDecode(conn *amqp.Connection) {
	// Declare queues using a dedicated channel
	ch, err := conn.Channel()
	log.Print("Message received ....", ch)
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Declare all necessary queues
	queues := []string{
		utils.AUTH_JWT_DECODE,
		utils.AUTH_JWT_DECODE_RESPONSE,
		utils.AUTH_JWT_DECODE_CONTACTS,
		utils.AUTH_JWT_DECODE_RESPONSE_CONTACTS,

		utils.AUTH_JWT_DECODE_MESSAGING,
		utils.AUTH_JWT_DECODE_RESPONSE_MESSAGING,
	}

	for _, queue := range queues {
		declareQueue(ch, queue)
	}
	log.Print("Declared all queues")
	// Start consumers with dedicated channels
	go jd.startConsumer(conn, utils.AUTH_JWT_DECODE, utils.AUTH_JWT_DECODE_RESPONSE)
	go jd.startConsumer(conn, utils.AUTH_JWT_DECODE_CONTACTS, utils.AUTH_JWT_DECODE_RESPONSE_CONTACTS)
	go jd.startConsumer(conn, utils.AUTH_JWT_DECODE_MESSAGING, utils.AUTH_JWT_DECODE_RESPONSE_MESSAGING)

	time.Sleep(2 * time.Second)

	log.Println("Listening for JWT decode requests...")
}

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
			log.Print("Decoded JWT", response)
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

// ListenForBatchDetails listens for batch requests and sends user details back
func ListenForBatchDetails(channel *amqp.Channel, userRepo repository.UserRepository) {
	// Declare the request queue
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
	log.Print("Messages ", msgs)
	// Process incoming requests
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
			log.Print("userDetails")
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
