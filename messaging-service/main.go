package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/streadway/amqp"
)

var JWT_SECRET []byte
var PORT, AMQP_URL string

// Validate JWT Token
func validateJWT(tokenString string) (bool, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return JWT_SECRET, nil
	})
	if err != nil {
		return false, err
	}
	return token.Valid, nil
}

// Handle message requests
func messageHandler(w http.ResponseWriter, r *http.Request) {

	// Validate JWT token from Authorization header
	token := r.Header.Get("Authorization")
	if token == "" {
		log.Println("Authorization header is missing")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Remove "Bearer " prefix
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	} else {
		log.Println("Authorization header does not contain Bearer token")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Validate the token
	if valid, err := validateJWT(token); !valid {
		log.Printf("Invalid token: %v", err)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Decode message payload
	var message map[string]string
	if err := json.NewDecoder(r.Body).Decode(&message); err != nil {
		log.Printf("Invalid request payload: %v", err)
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Publish message to RabbitMQ
	conn, err := amqp.Dial(AMQP_URL)
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Failed to open a channel: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("messages", false, false, false, false, nil)
	if err != nil {
		log.Printf("Failed to declare a queue: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = ch.Publish("", q.Name, false, false, amqp.Publishing{
		ContentType: "text/plain",
		Body:        []byte(message["content"]),
	})
	if err != nil {
		log.Printf("Failed to publish message: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	log.Printf("Message sent: %s", message["content"])
	w.WriteHeader(http.StatusOK)
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	PORT = os.Getenv("PORT")
	AMQP_URL = os.Getenv("AMQP_URL")
	JWT_SECRET = []byte(os.Getenv("JWT_SECRET"))

	http.HandleFunc("/send-message", messageHandler)
	log.Println("Messaging service running on port 8082")
	log.Fatal(http.ListenAndServe(":8082", nil))
}
