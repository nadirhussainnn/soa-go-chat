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
		log.Printf("[HandleLogin] Error communicating with Authentication Service: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var user struct {
		UserID       string `json:"user_id"`
		SessionToken string `json:"session_token"`
	}
	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		log.Printf("[HandleLogin] Failed to decode auth-service response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}
	log.Print("Logged in user", user)
	// Forward cookies from auth-service to the browser
	for _, cookie := range resp.Cookies() {
		http.SetCookie(w, cookie)
	}

	// Fetch only contacts from the contacts-service
	req, err := http.NewRequest("GET", GATEWAY_URL+"/contacts/?user_id="+user.UserID, nil)
	if err != nil {
		log.Printf("[HandleLogin] Failed to create request to contacts-service: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	req.AddCookie(&http.Cookie{
		Name:  "session_token",
		Value: user.SessionToken,
		Path:  "/",
	})

	client := &http.Client{}
	contactsResp, err := client.Do(req)
	if err != nil {
		log.Printf("[HandleLogin] Failed to fetch contacts for user %s: %v", user.UserID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer contactsResp.Body.Close()

	log.Print("Contacts response", contactsResp.Body)
	if contactsResp.StatusCode != http.StatusOK {
		log.Printf("[HandleLogin] Error from contacts-service for user %s: %s", user.UserID, contactsResp.Status)
		http.Error(w, "Failed to fetch contacts", http.StatusInternalServerError)
		return
	}

	var data struct {
		Contacts []struct {
			ID        string `json:"id"`
			UserID    string `json:"user_id"`
			ContactID string `json:"contact_id"`
			CreatedAt string `json:"created_at"`
			Details   struct {
				Username string `json:"username"`
				Email    string `json:"email"`
			} `json:"contactDetails"`
		} `json:"contacts"`
	}

	log.Print("Decoding contacts response", contactsResp.Body)

	err = json.NewDecoder(contactsResp.Body).Decode(&data)
	if err != nil {
		log.Printf("[HandleLogin] Failed to decode contacts response for user %s: %v", user.UserID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	log.Print("Decoded contacts response", data)
	// Pass contacts data to the template
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	err = tmpl.ExecuteTemplate(w, "dashboard.html", map[string]interface{}{
		"Contacts": data.Contacts,
		"UserID":   user.UserID,
	})

	if err != nil {
		log.Printf("[HandleLogin] Failed to render template for user %s: %v", user.UserID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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

func HandleContacts(w http.ResponseWriter, r *http.Request) {
	GATEWAY_URL := os.Getenv("GATEWAY_URL")
	log.Print("Gateway URL", GATEWAY_URL)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		log.Print("User ID not found in context")
		http.Error(w, "Unauthorized: user_id is required", http.StatusUnauthorized)
		return
	}

	cookie, err := r.Cookie("session_token")
	if err != nil || cookie.Value == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	log.Print("Session token", cookie.Value)
	// Fetch only contacts from the contacts-service
	req, err := http.NewRequest("GET", GATEWAY_URL+"/contacts/?user_id="+userID, nil)
	if err != nil {
		log.Printf("[HandleLogin] Failed to create request to contacts-service: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	req.AddCookie(cookie)

	log.Print("Request", req.Header)
	client := &http.Client{}
	contactsResp, err := client.Do(req)
	if err != nil {
		log.Printf("[HandleLogin] Failed to fetch contacts for user %s: %v", userID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer contactsResp.Body.Close()

	log.Print("Contacts response", contactsResp.Body)
	if contactsResp.StatusCode != http.StatusOK {
		log.Printf("[HandleLogin] Error from contacts-service for user %s: %s", userID, contactsResp.Status)
		http.Error(w, "Failed to fetch contacts", http.StatusInternalServerError)
		return
	}

	var data struct {
		Contacts []struct {
			ID        string `json:"id"`
			UserID    string `json:"user_id"`
			ContactID string `json:"contact_id"`
			CreatedAt string `json:"created_at"`
			Details   struct {
				Username string `json:"username"`
				Email    string `json:"email"`
			} `json:"contactDetails"`
		} `json:"contacts"`
	}

	log.Print("Decoding contacts response", contactsResp.Body)

	err = json.NewDecoder(contactsResp.Body).Decode(&data)
	if err != nil {
		log.Printf("[HandleLogin] Failed to decode contacts response for user %s: %v", userID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	log.Print("Decoded contacts response", data)
	// Pass contacts data to the template
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	err = tmpl.ExecuteTemplate(w, "dashboard.html", map[string]interface{}{
		"Contacts": data.Contacts,
		"UserID":   userID,
	})

	if err != nil {
		log.Printf("[HandleLogin] Failed to render template for user %s: %v", userID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
