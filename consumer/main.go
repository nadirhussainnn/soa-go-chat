package main

import (
	auth "consumer/handlers"
	"consumer/middleware"
	"consumer/utils"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

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
		utils.MESSAGES, // Queue name
		false,          // Durable
		false,          // Delete when unused
		false,          // Exclusive
		false,          // No-wait
		nil,            // Arguments
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

	go StartConsumer()

	// Use the RabbitMQ channel in middleware for session validation
	authMiddleware := &middleware.AuthMiddleware{
		AMQPChannel: ch, // Correctly pass the RabbitMQ channel

	}

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	tmpl := template.Must(template.ParseGlob("./templates/*.html"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "index.html", nil)
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tmpl.ExecuteTemplate(w, "login.html", nil)
		} else if r.Method == http.MethodPost {
			auth.HandleLogin(w, r)
		}
	})

	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tmpl.ExecuteTemplate(w, "register.html", nil)
		} else if r.Method == http.MethodPost {
			auth.HandleRegister(w, r)
		}
	})

	http.HandleFunc("/forgot-password", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tmpl.ExecuteTemplate(w, "forgot_password.html", nil)
		} else {
			auth.HandleForgotPassword(w, r)
		}
	})

	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		auth.HandleLogout(w, r)
	})

	http.HandleFunc("/terms", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "terms.html", nil)
	})

	http.Handle("/dashboard", authMiddleware.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value("user_id").(string)
		if !ok || userID == "" {
			log.Println("User ID not found in context")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Fetch contacts and contact requests as earlier
		GATEWAY_URL := os.Getenv("GATEWAY_URL")
		req, err := http.NewRequest("GET", GATEWAY_URL+"/contacts?user_id="+userID, nil)
		if err != nil {
			log.Printf("Failed to create request to contacts-service: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		cookie, _ := r.Cookie("session_token")
		req.AddCookie(cookie)

		client := &http.Client{
			Timeout: 10 * time.Second,
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Failed to fetch contacts: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Error from contacts-service: %s", resp.Status)
			http.Error(w, "Failed to fetch contacts", http.StatusInternalServerError)
			return
		}

		var data struct {
			Contacts []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"contacts"`
			ContactRequests []struct {
				SenderID   string `json:"sender_id"`
				ReceiverID string `json:"receiver_id"`
				Status     string `json:"status"`
			} `json:"contact_requests"`
		}

		err = json.NewDecoder(resp.Body).Decode(&data)
		if err != nil {
			log.Printf("Failed to decode contacts response: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		log.Printf("Fetched data for dashboard: %+v", data)

		tmpl.ExecuteTemplate(w, "dashboard.html", data)
	})))

	// Start the HTTP server
	log.Println("Consumer service running on port", PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, nil))
}
