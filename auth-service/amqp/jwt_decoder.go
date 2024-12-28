package amqp

import (
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
