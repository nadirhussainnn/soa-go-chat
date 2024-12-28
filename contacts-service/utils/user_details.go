package utils

import (
	"encoding/json"
	"errors"
	"log"

	"fmt"
	"net/http"

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
		return &response, nil
	}

	return nil, errors.New("no response from auth-service")
}

func GetUserDetails(authServiceURL, userID string) (*UserDetails, error) {
	url := fmt.Sprintf("%s/user/details?user_id=%s", authServiceURL, userID)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Failed to fetch user details: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Auth service returned non-OK status: %d", resp.StatusCode)
		return nil, fmt.Errorf("failed to fetch user details")
	}

	var userDetails UserDetails
	if err := json.NewDecoder(resp.Body).Decode(&userDetails); err != nil {
		log.Printf("Failed to decode user details response: %v", err)
		return nil, err
	}
	return &userDetails, nil
}
