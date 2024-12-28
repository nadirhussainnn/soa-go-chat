package auth

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"text/template"
)

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

	var user struct {
		SessionID    string `json:"session_id"`
		UserID       string `json:"user_id"`
		SessionToken string `json:"session_token"`
	}
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		log.Printf("Failed to decode auth-service response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode == http.StatusOK {
		// Forward cookies from auth-service to the browser
		for _, cookie := range resp.Cookies() {
			http.SetCookie(w, cookie)
		}

		// Fetch contacts from contacts-service
		req, err := http.NewRequest("GET", GATEWAY_URL+"/contacts/?user_id="+user.UserID, nil)
		if err != nil {
			log.Printf("Failed to create request to contacts-service: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		log.Printf("Forwarding request to contacts-service: %s", req.URL)

		// Set the session cookie
		cookie := &http.Cookie{
			Name:  "session_token",
			Value: user.SessionToken,
			Path:  "/",
		}

		req.AddCookie(cookie)
		client := &http.Client{}
		contactsResp, err := client.Do(req)
		if err != nil {
			log.Printf("Failed to fetch contacts: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		log.Print("Contacts response: ", contactsResp)
		defer contactsResp.Body.Close()

		if contactsResp.StatusCode != http.StatusOK {
			log.Printf("Error from contacts-service: %s", contactsResp.Status)
			http.Error(w, "Failed to fetch contacts", http.StatusInternalServerError)
			return
		}
		log.Print("Decoding contacts response")
		var data struct {
			Contacts []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"contacts"`
			ContactRequests []struct {
				ID                 string `json:"id"`
				SenderID           string `json:"sender_id"`
				ReceiverID         string `json:"receiver_id"`
				Status             string `json:"status"`
				CreatedAtFormatted string `json:"created_at_formatted"`
				SenderDetails      struct {
					Username string `json:"username"`
					Email    string `json:"email"`
				} `json:"sender_details"`
			} `json:"contactRequests"`
		}

		err = json.NewDecoder(contactsResp.Body).Decode(&data)
		if err != nil {
			log.Printf("Failed to decode contacts response: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		// Log contact requests properly
		for _, request := range data.ContactRequests {
			log.Printf("Contact Request - SenderID: %s, ReceiverID: %s, Status: %s", request.SenderID, request.ReceiverID, request.Status)
		}
		tmpl := template.Must(template.ParseGlob("templates/*.html"))
		err = tmpl.ExecuteTemplate(w, "dashboard.html", data)
		if err != nil {
			log.Printf("Failed to render template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
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
			Name:  "session_token",
			Value: "",
			// HttpOnly: true,
			Path:   "/",
			MaxAge: -1, // Expire immediately
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
