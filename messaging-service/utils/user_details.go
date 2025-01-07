// Handles sending AMQP request via RabbitMQ to auth-service to get user details from JWT
// Author: Nadir Hussain

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

type BatchDetailsRequest struct {
	UserIDs []string `json:"user_ids"`
}

// Sends the JWT to the auth-service via AMQP and retrieves user details.
// Params:
//   - amqpChannel: Pointer to the AMQP channel used for communication.
//   - sessionToken: The JWT session token to decode.
//
// Returns:
//   - *DecodeJWTResponse: Decoded JWT response containing user details.
//   - error: Any error encountered during the process.
func DecodeJWT(amqpChannel *amqp.Channel, sessionToken string) (*DecodeJWTResponse, error) {
	if sessionToken == "" {
		return nil, errors.New("session token is empty")
	}

	// Prepare the request payload
	request := DecodeJWTRequest{SessionToken: sessionToken}
	requestBytes, _ := json.Marshal(request)

	// Publish the request to the auth-service
	err := amqpChannel.Publish(
		"",                        // Exchange
		AUTH_JWT_DECODE_MESSAGING, // Routing key
		false,                     // Mandatory
		false,                     // Immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        requestBytes,
		},
	)
	if err != nil {
		log.Printf("Failed to publish JWT decode request: %v", err)
		return nil, err
	}
	// Consume the response from the auth-service
	msgs, err := amqpChannel.Consume(
		AUTH_JWT_DECODE_RESPONSE_MESSAGING, // Queue
		"",                                 // Consumer tag
		true,                               // Auto-acknowledge
		false,                              // Exclusive
		false,                              // No-local
		false,                              // No-wait
		nil,                                // Args
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
		return &response, nil
	}

	return nil, errors.New("no response from auth-service")
}
