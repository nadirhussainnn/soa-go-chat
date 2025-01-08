package middleware

import (
	"log"
	"net/http"
	"os"
)

func CorsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Print("Comed in middleware")
		// Allow requests from your frontend origin
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("*"))
		// Allow credentials (cookies, session tokens)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		// Allow specific HTTP methods
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, DELETE")
		// Allow specific headers
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests (OPTIONS)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		log.Print("hre Comed in middleware")

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}
