package amqp

import (
	"auth-service/models"
	"auth-service/repository"
	"auth-service/utils"
	"encoding/json"
	"log"

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

func (jd *JWTDecoder) ListenForJWTDecode(ch *amqp.Channel) {
	// Declare the request queue
	_, err := ch.QueueDeclare(
		utils.AUTH_JWT_DECODE, // Name of the request queue
		false,                 // Durable
		false,                 // Delete when unused
		false,                 // Exclusive
		false,                 // No-wait
		nil,                   // Arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare 'auth-jwt-decode' queue: %v", err)
	}

	// Declare the response queue
	_, err = ch.QueueDeclare(
		utils.AUTH_JWT_DECODE_RESPONSE, // Name of the response queue
		false,                          // Durable
		false,                          // Delete when unused
		false,                          // Exclusive
		false,                          // No-wait
		nil,                            // Arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare 'auth-jwt-decode-response' queue: %v", err)
	}

	msgs, err := ch.Consume(
		utils.AUTH_JWT_DECODE, // Queue name
		"",                    // Consumer tag
		true,                  // Auto-acknowledge
		false,                 // Exclusive
		false,                 // No-local
		false,                 // No-wait
		nil,                   // Args
	)
	if err != nil {
		log.Fatalf("Failed to register consumer for 'auth-jwt-decode': %v", err)
	}

	go func() {
		for d := range msgs {
			log.Printf("Received message body: %s", string(d.Body))

			var request DecodeJWTRequest
			v := json.Unmarshal(d.Body, &request)
			if v != nil {
				log.Printf("Failed to unmarshal JWT decode request: %v", v)
				continue
			}
			response := jd.decodeJWT(request.SessionToken)

			responseBytes, err := json.Marshal(response)
			if err != nil {
				log.Printf("Failed to marshal JWT decode response: %v", err)
				continue
			}

			// Publish response to the response queue
			err = ch.Publish(
				"",                             // Exchange
				utils.AUTH_JWT_DECODE_RESPONSE, // Routing key
				false,                          // Mandatory
				false,                          // Immediate
				amqp.Publishing{
					ContentType: "application/json",
					Body:        responseBytes,
				},
			)
			if err != nil {
				log.Printf("Failed to publish JWT decode response: %v", err)
			}
		}
	}()
	log.Println("Listening for JWT decode requests...")
}

func (jd *JWTDecoder) decodeJWT(token string) JWTDecodeResponse {

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

	// Process incoming requests
	go func() {
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

			err = channel.Publish(
				"",                                // Exchange
				utils.AUTH_BATCH_DETAILS_RESPONSE, // Routing key (response queue)
				false,                             // Mandatory
				false,                             // Immediate
				amqp.Publishing{
					ContentType: "application/json",
					Body:        responseBytes,
				},
			)
			if err != nil {
				log.Printf("Failed to publish response: %v", err)
			}
		}
	}()
	log.Println("Listening for batch details requests...")
}

// func ListenForBatchDetails(ch *amqp.Channel, userRepo repository.UserRepository) {
// 	// Declare request and response queues
// 	_, err := ch.QueueDeclare(
// 		utils.AUTH_BATCH_DETAILS_REQUEST, // Queue name
// 		false,                            // Durable
// 		false,                            // Delete when unused
// 		false,                            // Exclusive
// 		false,                            // No-wait
// 		nil,                              // Arguments
// 	)
// 	if err != nil {
// 		log.Fatalf("Failed to declare request queue: %v", err)
// 	}

// 	_, err = ch.QueueDeclare(
// 		utils.AUTH_BATCH_DETAILS_RESPONSE, // Queue name
// 		false,                             // Durable
// 		false,                             // Delete when unused
// 		false,                             // Exclusive
// 		false,                             // No-wait
// 		nil,                               // Arguments
// 	)
// 	if err != nil {
// 		log.Fatalf("Failed to declare response queue: %v", err)
// 	}

// 	// Consume messages from the request queue
// 	msgs, err := ch.Consume(
// 		utils.AUTH_BATCH_DETAILS_REQUEST, // Queue name
// 		"",                               // Consumer tag
// 		true,                             // Auto-acknowledge
// 		false,                            // Exclusive
// 		false,                            // No-local
// 		false,                            // No-wait
// 		nil,                              // Args
// 	)
// 	if err != nil {
// 		log.Fatalf("Failed to consume from queue: %v", err)
// 	}

// 	go func() {
// 		for d := range msgs {
// 			var request BatchDetailsRequest
// 			if err := json.Unmarshal(d.Body, &request); err != nil {
// 				log.Printf("Failed to unmarshal batch request: %v", err)
// 				continue
// 			}

// 			// Fetch user details
// 			userDetails := make(map[string]*models.SenderDetails)
// 			for _, userID := range request.UserIDs {
// 				user, err := userRepo.GetUserByID(userID)
// 				if err != nil {
// 					log.Printf("Failed to fetch user for user_id %s: %v", userID, err)
// 					continue
// 				}
// 				userDetails[userID] = &models.SenderDetails{
// 					Username: user.Username,
// 					Email:    user.Email,
// 				}
// 			}

// 			// Publish response
// 			response := BatchDetailsResponse{UserDetails: userDetails}
// 			responseBytes, err := json.Marshal(response)
// 			if err != nil {
// 				log.Printf("Failed to marshal response: %v", err)
// 				continue
// 			}

// 			err = ch.Publish(
// 				"",                                // Exchange
// 				utils.AUTH_BATCH_DETAILS_RESPONSE, // Routing key
// 				false,                             // Mandatory
// 				false,                             // Immediate
// 				amqp.Publishing{
// 					ContentType: "application/json",
// 					Body:        responseBytes,
// 				},
// 			)
// 			if err != nil {
// 				log.Printf("Failed to publish response: %v", err)
// 			}
// 		}
// 	}()
// 	log.Println("Listening for batch details requests...")
// }
