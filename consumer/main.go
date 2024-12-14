package main

import (
	auth "consumer/handlers"
	"consumer/utils"
	"log"
	"net/http"
	"os"
	"text/template"

	"github.com/streadway/amqp"
)

var (
	PORT, AMQP_URL string
	conn           *amqp.Connection
	ch             *amqp.Channel
)

// StartConsumer initializes the RabbitMQ consumer
func StartConsumer() {
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

	// Process messages in a goroutine
	go func() {
		for d := range msgs {
			log.Printf("Received message: %s", d.Body)
			// Add your processing logic here
		}
	}()
	log.Println("Consumer is running and listening for messages")
}

func main() {

	utils.LoadEnvs()

	PORT = os.Getenv("PORT")
	AMQP_URL = os.Getenv("AMQP_URL")

	// Connect to RabbitMQ
	conn, err := amqp.Dial(AMQP_URL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// Open a channel
	ch, err = conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Start the RabbitMQ consumer in a goroutine
	go StartConsumer()

	// Load templates
	templates := template.Must(template.ParseGlob("templates/*.html"))

	// Serve static files (CSS, JS, images, etc.)
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	tmpl := template.Must(template.ParseGlob("./templates/*.html"))

	// Serve the homepage
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "index.html", nil)
	})

	// Define HTTP handlers
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

	http.HandleFunc("/terms", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "terms.html", nil)
	})

	http.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "chat.html", nil)
	})

	http.HandleFunc("/forgot-password", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tmpl.ExecuteTemplate(w, "forgot_password.html", nil)
		} else {
			auth.HandleForgotPassword(w, r)
		}
	})

	// Start the HTTP server
	log.Println("Consumer service running on port", PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, nil))
}
