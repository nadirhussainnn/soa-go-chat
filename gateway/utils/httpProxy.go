package utils

import (
	"io"
	"log"
	"net/http"
)

func HttpProxyHandler(targetURL string, stripPrefix string) http.HandlerFunc {
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
