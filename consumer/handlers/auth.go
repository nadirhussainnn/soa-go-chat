package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// HandleLogin processes login requests
func HandleLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")
	GATEWAY_URL := os.Getenv("GATEWAY_URL")
	// Send data to Authentication Service
	payload := map[string]string{"username": username, "password": password}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(GATEWAY_URL+"/auth/login", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Error communicating with Authentication Service: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Forward the Set-Cookie header from auth-service to the browser
		for _, cookie := range resp.Cookies() {
			http.SetCookie(w, cookie)
		}

		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
	} else {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
	}
}

// HandleRegister processes registration requests
func HandleRegister(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")
	GATEWAY_URL := os.Getenv("GATEWAY_URL")
	// Send data to Authentication Service
	payload := map[string]string{"username": username, "email": email, "password": password}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(GATEWAY_URL+"/auth/register", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Error communicating with Authentication Service: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	} else {
		http.Error(w, "Registration failed", http.StatusBadRequest)
	}
}

// HandleLogin processes login requests
func HandleLogout(w http.ResponseWriter, r *http.Request) {
	GATEWAY_URL := os.Getenv("GATEWAY_URL")

	// Forward the session cookie to the auth-service
	client := &http.Client{}
	req, _ := http.NewRequest("POST", GATEWAY_URL+"/auth/logout", nil)

	// Include the session token from the cookie in the request
	cookie, err := r.Cookie("session_token")
	if err != nil {
		http.Error(w, "Session not found", http.StatusUnauthorized)
		return
	}
	req.AddCookie(cookie)

	// Send request to auth-service
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error communicating with Authentication Service: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Handle the response
	if resp.StatusCode == http.StatusOK {
		// Clear the session cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    "",
			HttpOnly: true,
			Path:     "/",
			MaxAge:   -1, // Expire immediately
		})

		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else {
		http.Error(w, "Failed to logout", http.StatusUnauthorized)
	}
}

// HandleForgotPassword processes forgot password requests
func HandleForgotPassword(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	newPassword := r.FormValue("new_password")
	GATEWAY_URL := os.Getenv("GATEWAY_URL")

	// Send data to Authentication Service
	payload := map[string]string{"username": username, "new_password": newPassword}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(GATEWAY_URL+"/auth/forgot-password", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Error communicating with Authentication Service: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	} else {
		http.Error(w, "Failed to reset password", http.StatusBadRequest)
	}
}

// HandleFetchAvailableContacts retrieves available contacts
func HandleFetchAvailableContacts(w http.ResponseWriter, r *http.Request) {
	GATEWAY_URL := os.Getenv("GATEWAY_URL")
	resp, err := http.Get(GATEWAY_URL + "/contacts/available")
	if err != nil {
		log.Printf("Error fetching available contacts: %v", err)
		http.Error(w, "Failed to fetch contacts", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(fmt.Sprintf("%s", resp.Body))); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

// HandleFetchMyContacts retrieves the logged-in user's contacts
func HandleFetchMyContacts(w http.ResponseWriter, r *http.Request) {
	GATEWAY_URL := os.Getenv("GATEWAY_URL")
	resp, err := http.Get(GATEWAY_URL + "/contacts/my")
	if err != nil {
		log.Printf("Error fetching user contacts: %v", err)
		http.Error(w, "Failed to fetch user contacts", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(fmt.Sprintf("%s", resp.Body))); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

// HandleSendContactRequest sends a contact request
func HandleSendContactRequest(w http.ResponseWriter, r *http.Request) {
	contactID := r.FormValue("contact_id")
	GATEWAY_URL := os.Getenv("GATEWAY_URL")
	payload := map[string]string{"contact_id": contactID}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(GATEWAY_URL+"/contacts/request", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Error sending contact request: %v", err)
		http.Error(w, "Failed to send request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		w.Write([]byte("Request sent successfully"))
	} else {
		http.Error(w, "Failed to send request", resp.StatusCode)
	}
}

// HandleRemoveContact removes a contact
func HandleRemoveContact(w http.ResponseWriter, r *http.Request) {
	contactID := r.FormValue("contact_id")
	GATEWAY_URL := os.Getenv("GATEWAY_URL")
	req, _ := http.NewRequest("DELETE", GATEWAY_URL+"/contacts/remove/"+contactID, nil)
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error removing contact: %v", err)
		http.Error(w, "Failed to remove contact", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		w.Write([]byte("Contact removed successfully"))
	} else {
		http.Error(w, "Failed to remove contact", resp.StatusCode)
	}
}
