package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

func proxyHandler(targetURL string, stripPrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// Strip prefix and construct target URL including query parameters
		forwardPath := r.URL.Path[len(stripPrefix):]
		fullURL := targetURL + forwardPath
		if r.URL.RawQuery != "" {
			fullURL += "?" + r.URL.RawQuery
		}

		// Create a new request with the same method, headers, and body
		client := &http.Client{}
		req, err := http.NewRequest(r.Method, fullURL, r.Body)
		if err != nil {
			log.Printf("Error creating request: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Copy headers from the original request to the new request
		for key, values := range r.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		// Copy cookies from the original request to the new request
		for _, cookie := range r.Cookies() {
			req.AddCookie(cookie)
		}
		log.Printf("Forwarding to: %s", fullURL)

		// Forward the request
		resp, err := client.Do(req)
		log.Print("Sending Request: ", req)
		if err != nil {
			log.Printf("Error forwarding request: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Copy response headers and status code to the original response writer
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		// Copy response body to the original response writer
		io.Copy(w, resp.Body)
		log.Printf("Response forwarded with status: %d", resp.StatusCode)
	}
}

func wsProxyHandler(serviceURLs map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Identify the target service based on the WebSocket path
		servicePath := mux.Vars(r)["path"]
		targetBaseURL, ok := serviceURLs[servicePath]
		if !ok {
			log.Printf("No target service found for path: %s", servicePath)
			http.Error(w, "Service not found", http.StatusNotFound)
			return
		}

		// Construct the target WebSocket URL, including query parameters
		targetURL := targetBaseURL
		if r.URL.RawQuery != "" {
			targetURL += "?" + r.URL.RawQuery
		}
		log.Printf("Forwarding WebSocket to: %s", targetURL)

		// Copy headers, excluding WebSocket-specific headers like `Connection` and `Upgrade`
		headers := http.Header{}
		for key, values := range r.Header {
			// Exclude WebSocket-specific headers
			if key == "Connection" || key == "Upgrade" || key == "Sec-Websocket-Key" || key == "Sec-Websocket-Extensions" || key == "Sec-Websocket-Version" {
				continue
			}
			for _, value := range values {
				headers.Add(key, value)
			}
		}

		// Establish a WebSocket connection to the target service
		targetConn, resp, err := websocket.DefaultDialer.Dial(targetURL, headers)
		if err != nil {
			log.Printf("Failed to connect to target WebSocket: %v", err)
			if resp != nil {
				body, _ := io.ReadAll(resp.Body)
				log.Printf("Response Status: %d, Body: %s", resp.StatusCode, string(body))
			}
			http.Error(w, "Failed to connect to backend WebSocket", http.StatusInternalServerError)
			return
		}
		defer targetConn.Close()

		// Upgrade the client connection to WebSocket
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow all origins for now (can be restricted based on security needs)
				return true
			},
		}
		clientConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Failed to upgrade client WebSocket: %v", err)
			return
		}
		defer clientConn.Close()

		// Proxy WebSocket messages between client and target service
		proxyWebSocket(clientConn, targetConn)
	}
}

// Function to proxy WebSocket messages
func proxyWebSocket(clientConn, targetConn *websocket.Conn) {
	// Forward messages from client to target
	go func() {
		for {
			messageType, message, err := clientConn.ReadMessage()
			if err != nil {
				log.Printf("Error reading from client WebSocket: %v", err)
				return
			}
			if err := targetConn.WriteMessage(messageType, message); err != nil {
				log.Printf("Error writing to target WebSocket: %v", err)
				return
			}
		}
	}()

	// Forward messages from target to client
	for {
		messageType, message, err := targetConn.ReadMessage()
		if err != nil {
			log.Printf("Error reading from target WebSocket: %v", err)
			return
		}
		if err := clientConn.WriteMessage(messageType, message); err != nil {
			log.Printf("Error writing to client WebSocket: %v", err)
			return
		}
	}
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found. Using environment variables instead.")
	}

	PORT := os.Getenv("PORT")
	AUTH_SERVICE_URL := os.Getenv("AUTH_SERVICE_URL")
	MESSAGING_SERVICE_URL := os.Getenv("MESSAGING_SERVICE_URL")
	CONTACTS_SERVICE_URL := os.Getenv("CONTACTS_SERVICE_URL")
	CONTACTS_SERVICE_WS_URL := os.Getenv("CONTACTS_SERVICE_WS_URL")
	MESSAGING_SERVICE_WS_URL := os.Getenv("MESSAGING_SERVICE_WS_URL")

	r := mux.NewRouter()
	r.HandleFunc("/auth/{path:.*}", proxyHandler(AUTH_SERVICE_URL, "/auth")).Methods("POST", "GET")
	r.HandleFunc("/contacts/{path:.*}", proxyHandler(CONTACTS_SERVICE_URL, "/contacts")).Methods("POST", "GET", "DELETE")
	r.HandleFunc("/messages/{path:.*}", proxyHandler(MESSAGING_SERVICE_URL, "/messages")).Methods("POST", "GET")

	// WebSocket Proxy
	serviceURLs := map[string]string{
		"contacts": CONTACTS_SERVICE_WS_URL,
		"messages": MESSAGING_SERVICE_WS_URL,
	}

	r.HandleFunc("/ws/{path}", wsProxyHandler(serviceURLs)).Methods("GET", "POST")

	log.Println("Gateway running on port", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprint(":", PORT), r))
}
