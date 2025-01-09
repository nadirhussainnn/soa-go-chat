// Entry point for the Consumer/Frontend-app. Initializes RabbitMQ, HTTP onnections with services
// Web sockets are handled within template files itself
// Author: Nadir Hussain

package main

import (
	"consumer/amqp"
	"io"

	handlers "consumer/handlers"
	"consumer/middleware"
	"consumer/utils"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

// Forward websocket request to gateway
func proxyWebSocket(dst, src *websocket.Conn, errChan chan error) {
	for {
		messageType, message, err := src.ReadMessage()
		if err != nil {
			errChan <- err
			return
		}

		err = dst.WriteMessage(messageType, message)
		if err != nil {
			errChan <- err
			return
		}
	}
}

func main() {

	// Loading environment variables from the .env file.
	utils.LoadEnvs()

	// Setting configuration variables.
	PORT := os.Getenv("PORT")
	AMQP_URL := os.Getenv("AMQP_URL")
	GATEWAY_WS_URL := os.Getenv("GATEWAY_WS_URL")
	GATEWAY_URL := os.Getenv("GATEWAY_URL")
	// Initializing RabbitMQ connection and channel.
	conn, _ := amqp.InitRabbitMQ(AMQP_URL) // Connection setup
	defer conn.Close()

	// Initializing middleware. As it needs connection to RabbitMQ
	authMiddleware := &middleware.AuthMiddleware{
		AMQPConn: conn,
	}

	// Configuring to server files from static folder
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Configuring to parse html templates
	tmpl := template.Must(template.ParseGlob("./templates/*.html"))

	// Defining routes for pagses, and handling Post requests along
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "index.html", nil)
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tmpl.ExecuteTemplate(w, "login.html", nil)
		} else if r.Method == http.MethodPost {
			handlers.HandleLogin(w, r)
		}
	})

	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tmpl.ExecuteTemplate(w, "register.html", nil)
		} else if r.Method == http.MethodPost {
			handlers.HandleRegister(w, r)
		}
	})

	http.HandleFunc("/forgot-password", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tmpl.ExecuteTemplate(w, "forgot_password.html", nil)
		} else {
			handlers.HandleForgotPassword(w, r)
		}
	})

	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleLogout(w, r)
	})

	http.HandleFunc("/terms", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "terms.html", nil)
	})

	// Handling protected routes by applying middleware
	http.Handle("/dashboard", authMiddleware.RequireAuth(http.HandlerFunc(handlers.HandleDashboard)))
	http.Handle("/contacts", authMiddleware.RequireAuth(http.HandlerFunc(handlers.HandleContacts)))
	http.Handle("/requests", authMiddleware.RequireAuth(http.HandlerFunc(handlers.HandleRequests)))
	http.Handle("/search", authMiddleware.RequireAuth(http.HandlerFunc(handlers.HandleSearch)))
	http.Handle("/messages", authMiddleware.RequireAuth(http.HandlerFunc(handlers.HandleMessages)))

	http.HandleFunc("/ws/", func(w http.ResponseWriter, r *http.Request) {
		// Build target URL to gateway
		targetURL := GATEWAY_WS_URL + r.URL.Path
		if r.URL.RawQuery != "" {
			targetURL += "?" + r.URL.RawQuery
		}

		log.Printf("Proxying WebSocket to: %s", targetURL)

		// Create upgrader
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}

		// Upgrade the client connection
		clientConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Failed to upgrade client connection: %v", err)
			return
		}
		defer clientConn.Close()

		// Connect to gateway
		gatewayConn, _, err := websocket.DefaultDialer.Dial(targetURL, nil)
		if err != nil {
			log.Printf("Failed to connect to gateway: %v", err)
			clientConn.Close()
			return
		}
		defer gatewayConn.Close()

		// Handle bidirectional communication
		errChan := make(chan error, 2)

		go proxyWebSocket(clientConn, gatewayConn, errChan)
		go proxyWebSocket(gatewayConn, clientConn, errChan)

		// Wait for an error
		<-errChan
	})
	// Start the HTTP server

	http.HandleFunc("/messages/", func(w http.ResponseWriter, r *http.Request) {

		targetURL := GATEWAY_URL + r.URL.Path

		if r.URL.RawQuery != "" {
			targetURL += "?" + r.URL.RawQuery
		}

		// Forward the session cookie
		req, err := http.NewRequest("GET", targetURL, nil)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}

		// Copy original cookies
		for _, cookie := range r.Cookies() {
			req.AddCookie(cookie)
		}

		// Make the request to gateway
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			http.Error(w, "Failed to download file", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Copy headers from gateway response
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		// Set content disposition header if it exists
		if contentDisposition := resp.Header.Get("Content-Disposition"); contentDisposition != "" {
			w.Header().Set("Content-Disposition", contentDisposition)
		}

		// Set content type if it exists
		if contentType := resp.Header.Get("Content-Type"); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}

		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
	log.Println("Consumer service running on port", PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, nil))
}
