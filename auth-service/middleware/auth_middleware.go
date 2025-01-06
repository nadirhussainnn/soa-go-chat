package middleware

import (
	"net/http"
)

// RequireAuth is the middleware for authenticating requests
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract session_token from the cookie
		cookie, err := r.Cookie("session_token")
		if err != nil || cookie.Value == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Proceed to the next handler if the token is present
		next.ServeHTTP(w, r)
	})
}
