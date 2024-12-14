package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func proxyHandler(targetURL string, stripPrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Strip prefix and construct target URL
		forwardPath := r.URL.Path[len(stripPrefix):]
		fullURL := targetURL + forwardPath

		// Create a new request with the same method, headers, and body
		client := &http.Client{}
		req, err := http.NewRequest(r.Method, fullURL, r.Body)
		if err != nil {
			log.Printf("Error creating request: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Copy headers from original request to new request
		for key, values := range r.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		resp, err := client.Do(req)
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

func main() {
	r := mux.NewRouter()

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	PORT := os.Getenv("PORT")
	AUTH_SERVICE_URL := os.Getenv("AUTH_SERVICE_URL")
	MESSAGING_SERVICE_URL := os.Getenv("MESSAGING_SERVICE_URL")
	// NOTIFICATION_SERVICE_URL := os.Getenv("NOTIFICATION_SERVICE_URL")
	// FRONTEND_URL := os.Getenv("FRONTEND_URL")

	r.HandleFunc("/auth/{path:.*}", proxyHandler(AUTH_SERVICE_URL, "/auth")).Methods("POST", "GET")
	r.HandleFunc("/message/{path:.*}", proxyHandler(MESSAGING_SERVICE_URL, "/message")).Methods("POST", "GET")

	log.Println("Gateway running on port", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprint(":", PORT), r))
}
