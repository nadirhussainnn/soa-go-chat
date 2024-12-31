// package utils

// import (
// 	"encoding/json"
// 	"errors"
// 	"log"

// 	"github.com/google/uuid"
// 	"github.com/streadway/amqp"
// )

// type DecodeJWTRequest struct {
// 	SessionToken string `json:"session_token"`
// }

// type DecodeJWTResponse struct {
// 	Valid    bool   `json:"valid"`
// 	UserID   string `json:"user_id"`
// 	Email    string `json:"email"`
// 	Username string `json:"username"`
// 	Error    string `json:"error,omitempty"`
// }

// // DecodeJWT sends the JWT to the auth-service via AMQP and retrieves user details
// func DecodeJWT(amqpChannel *amqp.Channel, sessionToken string) (*DecodeJWTResponse, error) {
// 	if sessionToken == "" {
// 		return nil, errors.New("session token is empty")
// 	}

// 	// Prepare the request payload
// 	request := DecodeJWTRequest{SessionToken: sessionToken}
// 	requestBytes, _ := json.Marshal(request)

// 	correlationID := uuid.New().String() // Generate a unique CorrelationId
// 	log.Print("Sending with corrrr ", correlationID)
// 	// Publish the request to the auth-service
// 	err := amqpChannel.Publish(
// 		"",              // Exchange
// 		AUTH_JWT_DECODE, // Routing key
// 		false,           // Mandatory
// 		false,           // Immediate
// 		amqp.Publishing{
// 			ContentType:   "application/json",
// 			Body:          requestBytes,
// 			CorrelationId: correlationID,
// 		},
// 	)
// 	if err != nil {
// 		log.Printf("Failed to publish JWT decode request: %v", err)
// 		return nil, err
// 	}
// 	log.Print("Sent")
// 	// Consume the response from the auth-service
// 	msgs, err := amqpChannel.Consume(
// 		AUTH_JWT_DECODE_RESPONSE, // Queue
// 		"",                       // Consumer tag
// 		true,                     // Auto-acknowledge
// 		false,                    // Exclusive
// 		false,                    // No-local
// 		false,                    // No-wait
// 		nil,                      // Args
// 	)
// 	if err != nil {
// 		log.Printf("Failed to consume JWT decode response: %v", err)
// 		return nil, err
// 	}

// 	log.Print("Received response", msgs)
// 	// Wait for the response
// 	for d := range msgs {
// 		var response DecodeJWTResponse
// 		if err := json.Unmarshal(d.Body, &response); err != nil {
// 			log.Printf("Failed to unmarshal JWT decode response: %v", err)
// 			continue
// 		}
// 		return &response, nil
// 	}

// 	return nil, errors.New("no response from auth-service")
// }

package utils

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/streadway/amqp"
)

type DecodeJWTRequest struct {
	SessionToken string `json:"session_token"`
}

type UserDetails struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

type DecodeJWTResponse struct {
	Valid    bool   `json:"valid"`
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Error    string `json:"error,omitempty"`
}

// DecodeJWT sends the JWT to the auth-service via AMQP and retrieves user details
func DecodeJWT(amqpChannel *amqp.Channel, sessionToken string) (*DecodeJWTResponse, error) {
	if sessionToken == "" {
		return nil, errors.New("session token is empty")
	}

	// Prepare the request payload
	request := DecodeJWTRequest{SessionToken: sessionToken}
	requestBytes, _ := json.Marshal(request)

	// Publish the request to the auth-service
	err := amqpChannel.Publish(
		"",              // Exchange
		AUTH_JWT_DECODE, // Routing key
		false,           // Mandatory
		false,           // Immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        requestBytes,
		},
	)
	if err != nil {
		log.Printf("Failed to publish JWT decode request: %v", err)
		return nil, err
	}
	log.Print("Published JWT decode request")
	// Consume the response from the auth-service
	msgs, err := amqpChannel.Consume(
		AUTH_JWT_DECODE_RESPONSE, // Queue
		"",                       // Consumer tag
		true,                     // Auto-acknowledge
		false,                    // Exclusive
		false,                    // No-local
		false,                    // No-wait
		nil,                      // Args
	)
	if err != nil {
		log.Printf("Failed to consume JWT decode response: %v", err)
		return nil, err
	}

	// Wait for the response
	for d := range msgs {
		var response DecodeJWTResponse
		if err := json.Unmarshal(d.Body, &response); err != nil {
			log.Printf("Failed to unmarshal JWT decode response: %v", err)
			continue
		}
		log.Print("Message: ", d.Body)
		return &response, nil
	}

	return nil, errors.New("no response from auth-service")
}
