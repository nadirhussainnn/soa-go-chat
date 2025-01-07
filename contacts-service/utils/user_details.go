// Handles sending AMQP request via RabbitMQ to auth-service to get user details from JWT
// Author: Nadir Hussain

package utils

import (
	"contacts-service/models"
	"encoding/json"
	"errors"
	"log"

	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

// The request sent to auth-service
type DecodeJWTRequest struct {
	SessionToken string `json:"session_token"`
}

// Each user detail contains this structured data
type UserDetails struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// The response of decoded JWt received in queue
type DecodeJWTResponse struct {
	Valid    bool   `json:"valid"`
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Error    string `json:"error,omitempty"`
}

// Structure of request for users' details to be sent to auth-service via amqp using rabbitmq broker
type BatchDetailsRequest struct {
	UserIDs []string `json:"user_ids"`
}

// The response published to queue by auth-service with users' details
type BatchDetailsResponse struct {
	UserDetails map[string]*models.SenderDetails `json:"user_details"`
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
		"",                       // Exchange
		AUTH_JWT_DECODE_CONTACTS, // Routing key
		false,                    // Mandatory
		false,                    // Immediate
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
		AUTH_JWT_DECODE_RESPONSE_CONTACTS, // Queue
		"",                                // Consumer tag
		true,                              // Auto-acknowledge
		false,                             // Exclusive
		false,                             // No-local
		false,                             // No-wait
		nil,                               // Args
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

// GetUsersDetails sends a batch request to auth-service and retrieves user details
func GetUsersDetails(channel *amqp.Channel, userIDs []string) (map[string]*models.SenderDetails, error) {
	// Prepare the request payload
	request := BatchDetailsRequest{UserIDs: userIDs}
	payload, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	// Declare the response queue (static, pre-defined)
	_, err = channel.QueueDeclare(
		AUTH_BATCH_DETAILS_RESPONSE, // Queue name (use a constant)
		false,                       // Durable
		false,                       // Delete when unused
		false,                       // Exclusive
		false,                       // No-wait
		nil,                         // Arguments
	)
	if err != nil {
		log.Printf("Failed to declare response queue: %v", err)
		return nil, err
	}

	correlationID := uuid.New().String() // Generate a unique CorrelationId

	// Publish the request to the request queue
	err = channel.Publish(
		"",                         // Exchange
		AUTH_BATCH_DETAILS_REQUEST, // Routing key (request queue)
		false,                      // Mandatory
		false,                      // Immediate
		amqp.Publishing{
			ContentType:   "application/json",
			Body:          payload,
			CorrelationId: correlationID,
		},
	)
	if err != nil {
		log.Printf("Failed to publish batch request: %v", err)
		return nil, err
	}
	log.Print("Sent request with coooorr", correlationID)
	// Consume messages from the response queue
	msgs, err := channel.Consume(
		AUTH_BATCH_DETAILS_RESPONSE, // Queue name (response queue)
		"",                          // Consumer tag
		true,                        // Auto-acknowledge
		false,                       // Exclusive
		false,                       // No-local
		false,                       // No-wait
		nil,                         // Args
	)
	if err != nil {
		log.Printf("Failed to consume from response queue: %v", err)
		return nil, err
	}

	log.Print("Received msgs from queue", msgs)
	// Wait for a response
	for msg := range msgs {
		var response BatchDetailsResponse
		err = json.Unmarshal(msg.Body, &response)
		if err != nil {
			log.Printf("Failed to unmarshal response: %v", err)
			continue
		}
		return response.UserDetails, nil
	}

	return nil, errors.New("no response received from auth-service")
}
