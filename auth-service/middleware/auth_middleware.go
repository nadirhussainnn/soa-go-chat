package middleware

import (
	"auth-service/utils"
	"context"
	"log"
	"net/http"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

type JWTDecodeResponse struct {
	UserID   string `json:"user_id,omitempty"`
	Username string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
	Valid    bool   `json:"valid"`
	Error    string `json:"error,omitempty"`
}

type JWTDecoder struct {
	Secret string
}

func (jd *JWTDecoder) DecodeJWT(token string) JWTDecodeResponse {

	tkn, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(jd.Secret), nil
	})

	if err != nil || !tkn.Valid {
		return JWTDecodeResponse{
			Valid: false,
			Error: err.Error(),
		}
	}

	claims, ok := tkn.Claims.(jwt.MapClaims)
	if !ok {
		return JWTDecodeResponse{
			Valid: false,
			Error: "Invalid claims structure",
		}
	}

	return JWTDecodeResponse{
		Valid:    true,
		UserID:   claims["id"].(string),
		Username: claims["username"].(string),
		Email:    claims["email"].(string),
	}
}

// RequireAuth is the middleware for authenticating requests
func RequireAuth(next http.Handler) http.Handler {

	utils.LoadEnvs()
	JWT_SECRET := os.Getenv("JWT_SECRET")
	jwtDecoder := &JWTDecoder{
		Secret: JWT_SECRET, // Replace with your actual secret key
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract session_token from the cookie
		cookie, err := r.Cookie("session_token")
		if err != nil || cookie.Value == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Decode the JWT using the JWTDecoder instance
		response := jwtDecoder.DecodeJWT(cookie.Value)
		log.Print("Decode Response", response)

		if !response.Valid {
			log.Printf("Invalid session: %s", response.Error)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		log.Print("Passing [user] context to route", response.UserID, response.Username, response.Email)

		// Extract user_id from session token and add it to request context
		ctx := context.WithValue(r.Context(), "user_id", response.UserID)
		ctx = context.WithValue(ctx, "username", response.Username)
		ctx = context.WithValue(ctx, "email", response.Email)

		next.ServeHTTP(w, r.WithContext(ctx))

		// Proceed to the next handler if the token is present
		// next.ServeHTTP(w, r)
	})
}
