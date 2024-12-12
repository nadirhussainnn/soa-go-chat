package auth

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

// HandleLogin processes login requests
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Send data to Authentication Service
	payload := map[string]string{"username": username, "password": password}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post("http://localhost:8080/auth/login", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Error communicating with Authentication Service: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
	}
}

// HandleRegister processes registration requests
func HandleRegister(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Send data to Authentication Service
	payload := map[string]string{"username": username, "password": password}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post("http://localhost:8080/auth/register", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Error communicating with Authentication Service: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	} else {
		http.Error(w, "Registration failed", http.StatusBadRequest)
	}
}
