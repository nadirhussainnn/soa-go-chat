package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secretKey = []byte("my-secret-key")

func generateJWT(username string) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(time.Hour * 1).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {

	var credentials map[string]string
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		log.Printf("Invalid request payload: %v", err) // Log the actual error

		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if credentials["username"] == "user" && credentials["password"] == "password" {
		token, _ := generateJWT(credentials["username"])
		json.NewEncoder(w).Encode(map[string]string{"token": token})
	} else {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	}
}

func main() {
	http.HandleFunc("/login", loginHandler)
	log.Println("Auth service running on port 8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}
