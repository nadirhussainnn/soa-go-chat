package main

import (
	auth "consumer/handlers"
	"log"
	"net/http"
	"text/template"

	"github.com/streadway/amqp"
)

func StartConsumer() {
	// Connect to RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// Open a channel
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Declare the queue
	q, err := ch.QueueDeclare(
		"messages", // Queue name
		false,      // Durable
		false,      // Delete when unused
		false,      // Exclusive
		false,      // No-wait
		nil,        // Arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	// Consume messages
	msgs, err := ch.Consume(
		q.Name, // Queue name
		"",     // Consumer
		true,   // Auto-acknowledge
		false,  // Exclusive
		false,  // No-local
		false,  // No-wait
		nil,    // Args
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	// Process messages
	go func() {
		for d := range msgs {
			log.Printf("Received message: %s", d.Body)
			// Add your processing logic here
		}
	}()
	log.Println("Consumer is running and listening for messages")
}

func main() {
	log.Println("Message Consumer Service started")

	// Start the RabbitMQ consumer in a goroutine
	go StartConsumer()

	// Load templates
	templates := template.Must(template.ParseGlob("templates/*.html"))

	// Define handlers
	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			templates.ExecuteTemplate(w, "login.html", nil)
		} else if r.Method == http.MethodPost {
			auth.HandleLogin(w, r)
		}
	})

	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			templates.ExecuteTemplate(w, "register.html", nil)
		} else if r.Method == http.MethodPost {
			auth.HandleRegister(w, r)
		}
	})

	// Start the HTTP server
	log.Println("Consumer service running on port 8085")
	log.Fatal(http.ListenAndServe(":8085", nil))
}
