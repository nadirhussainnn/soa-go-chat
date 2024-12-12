package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func proxyHandler(targetURL string, stripPrefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Strip prefix and construct target URL
		forwardPath := r.URL.Path[len(stripPrefix):]
		fullURL := targetURL + forwardPath

		log.Printf("Forwarding request %s %s to %s", r.Method, r.URL.Path, fullURL)

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

		// Forward the request and handle the response
		fmt.Print(req)
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
				log.Printf("Received Authorization header: %s", value)
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

	r.HandleFunc("/auth/{path:.*}", proxyHandler("http://127.0.0.1:8081", "/auth")).Methods("POST", "GET")
	r.HandleFunc("/message/{path:.*}", proxyHandler("http://127.0.0.1:8082", "/message")).Methods("POST", "GET")

	log.Println("Gateway running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
