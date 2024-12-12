package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

var PORT, JWT_SECRET string

func generateJWT(username string) (string, error) {
	fmt.Print("JWT_SECRET")
	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(time.Hour * 1).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWT_SECRET)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {

	var credentials map[string]string
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		log.Printf("Invalid request payload: %v", err)

		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if credentials["username"] == "user" && credentials["password"] == "password" {
		log.Println("Lovking....1", credentials["username"])
		token, _ := generateJWT(credentials["username"])
		log.Println("Lovking....token", token)
		json.NewEncoder(w).Encode(map[string]string{"token": token})
		log.Println("Lovking....3")
	} else {
		log.Println("Lovking....4")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		log.Println("Lovking....5")
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	PORT = os.Getenv("PORT")
	JWT_SECRET = os.Getenv("JWT_SECRET")
	http.HandleFunc("/login", loginHandler)
	log.Println("Auth service running on port", PORT, JWT_SECRET)
	log.Fatal(http.ListenAndServe(fmt.Sprint(":", PORT), nil))
}
