package utils

import (
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

func WebSocketProxyHandler(serviceURLs map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Identify the target service based on the WebSocket path
		log.Print("Comes here")
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
