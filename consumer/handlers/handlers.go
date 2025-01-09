// Provides handlers to consume auth-service, contacts-service and messaging-serive APIs and Queues
// Author: Nadir Hussain

package handlers

import (
	"bytes"
	"consumer/utils"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)

// Defines the structure for user related data
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// Defines the structure for message related data
type Message struct {
	ID           string `json:"id"`
	SenderID     string `json:"sender_id"`
	ReceiverID   string `json:"receiver_id"`
	Content      string `json:"content"`
	CreatedAt    string `json:"created_at"`
	MessageType  string `json:"message_type"` // 'text', 'file'
	FilePath     string `json:"file_path"`    // Path to the file on the server
	FileName     string `json:"file_name"`    // Name of the file
	FileMimeType string `json:"file_mime_type"`
}

// Processes the login request from the user and fetches user contacts.
// Params:
//   - w: http.ResponseWriter to write the response.
//   - r: *http.Request containing the request data.
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
		utils.RenderLoginWithError(w, "Internal Server Error")

		return
	}
	defer resp.Body.Close()

	var user struct {
		UserID       string `json:"user_id"`
		SessionToken string `json:"session_token"`
		UserName     string `json:"username"`
		Email        string `json:"email"`
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil || errorResponse.Message == "" {
			utils.RenderLoginWithError(w, "Invalid username or password")
		} else {
			utils.RenderLoginWithError(w, errorResponse.Message)
		}
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&user)
	if err != nil {
		log.Printf("[HandleLogin] Failed to decode auth-service response: %v", err)
		utils.RenderLoginWithError(w, "Internal Server Error")
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
		utils.RenderLoginWithError(w, "Internal Server Error")
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
		utils.RenderLoginWithError(w, "Failed to fetch contacts")
		return
	}
	defer contactsResp.Body.Close()

	log.Print("Contacts response", contactsResp.Body)
	if contactsResp.StatusCode != http.StatusOK {
		log.Printf("[HandleLogin] Error from contacts-service for user %s: %s", user.UserID, contactsResp.Status)
		utils.RenderLoginWithError(w, "Failed to fetch contacts")
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
				UserID   string `json:"user_id"`
			} `json:"contactDetails"`
		} `json:"contacts"`
	}

	err = json.NewDecoder(contactsResp.Body).Decode(&data)
	if err != nil {
		log.Printf("[HandleLogin] Failed to decode contacts response for user %s: %v", user.UserID, err)
		utils.RenderLoginWithError(w, "Internal Server Error")
		return
	}
	// Pass contacts data to the template
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	err = tmpl.ExecuteTemplate(w, "dashboard.html", map[string]interface{}{
		"Contacts":       data.Contacts,
		"UserID":         user.UserID,
		"Username":       user.UserName,
		"Email":          user.Email,
		"WebSocketURL":   fmt.Sprintf("ws://%s", r.Host), // Use the host from the request
		"GatewayHttpURL": fmt.Sprintf("http://%s", r.Host),
		// "WebSocketURL":   os.Getenv("GATEWAY_WS_URL"),
		// "GatewayHttpURL": os.Getenv("GATEWAY_URL"),
	})

	if err != nil {
		log.Printf("[HandleLogin] Failed to render template for user %s: %v", user.UserID, err)
		utils.RenderLoginWithError(w, "Internal Server Error")
	}
}

// Processes the user registration request.
// Params:
//   - w: http.ResponseWriter to write the response.
//   - r: *http.Request containing the request data.
func HandleRegister(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")
	GATEWAY_URL := os.Getenv("GATEWAY_URL")

	// Performing validation on input data
	if err := utils.ValidateRegistrationInput(username, email, password); err != nil {
		utils.RenderRegisterWithError(w, err.Error())
		return
	}

	// Send data to Authentication Service
	payload := map[string]string{"username": username, "email": email, "password": password}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(GATEWAY_URL+"/auth/register", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Error communicating with Authentication Service: %v", err)
		utils.RenderRegisterWithError(w, "Internal Server Error")
		return
	}
	defer resp.Body.Close()

	// Check for specific error responses from the backend
	if resp.StatusCode == http.StatusConflict {
		utils.RenderRegisterWithError(w, "User with this username or email already exists")
		return
	}

	if resp.StatusCode != http.StatusCreated {
		utils.RenderRegisterWithError(w, "Registration failed. Please try again.")
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)

}

// Processes user logout requests by invalidating the session.
// Params:
//   - w: http.ResponseWriter to send the response.
//   - r: *http.Request containing the logout request.
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

// Handles the password reset process for users
// Params:
//   - w: http.ResponseWriter to send the response.
//   - r: *http.Request containing the request data.
func HandleForgotPassword(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	newPassword := r.FormValue("new_password")
	GATEWAY_URL := os.Getenv("GATEWAY_URL")

	// Performing validation on input data
	if err := utils.ValidatePassword(newPassword); err != nil {
		utils.RenderResetPassWithError(w, err.Error())
		return
	}

	// Send data to Authentication Service
	payload := map[string]string{"username": username, "new_password": newPassword}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(GATEWAY_URL+"/auth/forgot-password", "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Error communicating with Authentication Service: %v", err)
		utils.RenderResetPassWithError(w, "Internal Server Error")
		return
	}
	defer resp.Body.Close()

	// Check for specific error responses from the backend
	if resp.StatusCode == http.StatusConflict {
		utils.RenderRegisterWithError(w, "User with this username or email does not exists")
		return
	}

	if resp.StatusCode != http.StatusOK {
		utils.RenderResetPassWithError(w, "Failed to reset password. Please try again.")
		return
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// Fetches and displays a user's contacts
// Params:
//   - w: http.ResponseWriter to send the response.
//   - r: *http.Request containing the request data.
func HandleContacts(w http.ResponseWriter, r *http.Request) {
	GATEWAY_URL := os.Getenv("GATEWAY_URL")
	log.Print("Gateway URL", GATEWAY_URL)
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		log.Print("User ID not found in context")
		http.Error(w, "Unauthorized: user_id is required", http.StatusUnauthorized)
		return
	}

	username, ok := r.Context().Value("username").(string)
	if !ok || userID == "" {
		log.Print("Username not found in context")
		http.Error(w, "Unauthorized: username is required", http.StatusUnauthorized)
		return
	}

	email, ok := r.Context().Value("email").(string)
	if !ok || userID == "" {
		log.Print("Email not found in context")
		http.Error(w, "Unauthorized: email is required", http.StatusUnauthorized)
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
				UserID   string `json:"user_id"`
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

	// JSON encode contacts for embedding in template
	contactsJSON, err := json.Marshal(data.Contacts)
	if err != nil {
		log.Printf("[HandleContacts] Failed to encode contacts as JSON: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Pass contacts data to the template
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	err = tmpl.ExecuteTemplate(w, "contacts.html", map[string]interface{}{
		"Contacts":     data.Contacts,
		"ContactsJSON": template.JS(contactsJSON), // Safe JSON for embedding
		"UserID":       userID,
		"Username":     username,
		"Email":        email,
		"WebSocketURL": fmt.Sprintf("ws://%s", r.Host), // Use the host from the request

		// "WebSocketURL": os.Getenv("GATEWAY_WS_URL"),
	})

	if err != nil {
		log.Printf("[HandleLogin] Failed to render template for user %s: %v", userID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Fetches and displays a user's contact requests
// Params:
//   - w: http.ResponseWriter to send the response.
//   - r: *http.Request containing the request data.
func HandleRequests(w http.ResponseWriter, r *http.Request) {
	GATEWAY_URL := os.Getenv("GATEWAY_URL")
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		log.Print("User ID not found in context")
		http.Error(w, "Unauthorized: user_id is required", http.StatusUnauthorized)
		return
	}

	username, ok := r.Context().Value("username").(string)
	if !ok || userID == "" {
		log.Print("Username not found in context")
		http.Error(w, "Unauthorized: username is required", http.StatusUnauthorized)
		return
	}

	email, ok := r.Context().Value("email").(string)
	if !ok || userID == "" {
		log.Print("Email not found in context")
		http.Error(w, "Unauthorized: email is required", http.StatusUnauthorized)
		return
	}

	cookie, err := r.Cookie("session_token")
	if err != nil || cookie.Value == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Fetch contact requests from the contacts-service
	req, err := http.NewRequest("GET", GATEWAY_URL+"/contacts/requests/?user_id="+userID, nil)
	if err != nil {
		log.Printf("[HandleRequests] Failed to create request to requests-service: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	req.AddCookie(cookie)

	client := &http.Client{}
	requestsResp, err := client.Do(req)
	if err != nil {
		log.Printf("[HandleRequests] Failed to fetch requests for user %s: %v", userID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer requestsResp.Body.Close()

	if requestsResp.StatusCode != http.StatusOK {
		log.Printf("[HandleRequests] Error from requests-service for user %s: %s", userID, requestsResp.Status)
		http.Error(w, "Failed to fetch requests", http.StatusInternalServerError)
		return
	}

	// Decode the response JSON
	var requests []struct {
		ID            string `json:"id"`
		SenderDetails struct {
			Username string `json:"username"`
			Email    string `json:"email"`
			UserID   string `json:"user_id"`
		} `json:"sender_details"`
		CreatedAtFormatted string `json:"created_at_formatted"`
	}

	err = json.NewDecoder(requestsResp.Body).Decode(&requests)
	if err != nil {
		log.Printf("[HandleRequests] Failed to decode requests response for user %s: %v", userID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("Decoded requests response for user %s: %+v", userID, requests)

	// Render the requests using the `requests.html` template
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	err = tmpl.ExecuteTemplate(w, "requests.html", map[string]interface{}{
		"Requests":     requests,
		"UserID":       userID,
		"Username":     username,
		"Email":        email,
		"WebSocketURL": fmt.Sprintf("ws://%s", r.Host), // Use the host from the request

		// "WebSocketURL": os.Getenv("GATEWAY_WS_URL"),
	})
	if err != nil {
		log.Printf("[HandleRequests] Failed to render template for user %s: %v", userID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Renders the dashboard page with user and contact details
// Params:
//   - w: http.ResponseWriter to send the response.
//   - r: *http.Request containing the request data.
func HandleDashboard(w http.ResponseWriter, r *http.Request) {
	GATEWAY_URL := os.Getenv("GATEWAY_URL")

	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		log.Print("User ID not found in context")
		http.Error(w, "Unauthorized: user_id is required", http.StatusUnauthorized)
		return
	}

	username, ok := r.Context().Value("username").(string)
	if !ok || userID == "" {
		log.Print("Username not found in context")
		http.Error(w, "Unauthorized: username is required", http.StatusUnauthorized)
		return
	}

	email, ok := r.Context().Value("email").(string)
	if !ok || userID == "" {
		log.Print("Email not found in context")
		http.Error(w, "Unauthorized: email is required", http.StatusUnauthorized)
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
				UserID   string `json:"user_id"`
			} `json:"contactDetails"`
		} `json:"contacts"`
	}

	err = json.NewDecoder(contactsResp.Body).Decode(&data)
	if err != nil {
		log.Printf("[HandleLogin] Failed to decode contacts response for user %s: %v", userID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Pass contacts data to the template
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	err = tmpl.ExecuteTemplate(w, "dashboard.html", map[string]interface{}{
		"Contacts":       data.Contacts,
		"UserID":         userID,
		"Username":       username,
		"Email":          email,
		"WebSocketURL":   fmt.Sprintf("ws://%s", r.Host), // Use the host from the request
		"GatewayHttpURL": fmt.Sprintf("http://%s", r.Host),

		// "WebSocketURL":   os.Getenv("GATEWAY_WS_URL"),
		// "GatewayHttpURL": os.Getenv("GATEWAY_URL"),
	})

	if err != nil {
		log.Printf("[HandleLogin] Failed to render template for user %s: %v", userID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Fetches and returns search results for users
// Params:
//   - w: http.ResponseWriter to send the response.
//   - r: *http.Request containing the request data.
func HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "Query parameter is required", http.StatusBadRequest)
		return
	}

	// Call the auth-service search API
	gatewayURL := os.Getenv("GATEWAY_URL")
	searchURL := gatewayURL + "/auth/search?q=" + query

	cookie, err := r.Cookie("session_token")
	if err != nil || cookie.Value == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Fetch only contacts from the contacts-service
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		log.Printf("[HandleLogin] Failed to create request to auth-service: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	req.AddCookie(cookie)

	log.Print("Request", req.Header)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[HandleLogin] Failed to search contacts %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Search API returned non-OK status: %v", resp.Status)
		http.Error(w, "Failed to fetch search results", http.StatusInternalServerError)
		return
	}

	// Parse the response
	var users []User // Match the API's array structure
	err = json.NewDecoder(resp.Body).Decode(&users)
	if err != nil {
		log.Printf("Failed to parse search response: %v", err)
		http.Error(w, "Failed to parse search results", http.StatusInternalServerError)
		return
	}

	// Render results as JSON
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(users)
	if err != nil {
		log.Printf("Failed to render search results: %v", err)
		http.Error(w, "Failed to render search results", http.StatusInternalServerError)
	}
}

// HandleMessages fetches messages between the logged-in user and a selected contact
// Params:
//   - w: http.ResponseWriter to send the response.
//   - r: *http.Request containing the request data.
func HandleMessages(w http.ResponseWriter, r *http.Request) {
	GATEWAY_URL := os.Getenv("GATEWAY_URL")
	userID := r.URL.Query().Get("user_id")
	contactID := r.URL.Query().Get("contact_id")

	// Check if user_id and contact_id are provided
	if userID == "" || contactID == "" {
		http.Error(w, "Missing user_id or contact_id", http.StatusBadRequest)
		return
	}

	// Extract session cookie
	cookie, err := r.Cookie("session_token")
	if err != nil || cookie.Value == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Make a request to the messaging service
	req, err := http.NewRequest("GET", GATEWAY_URL+"/messages/?user_id="+userID+"&contact_id="+contactID, nil)
	if err != nil {
		log.Printf("[HandleMessages] Failed to create request to messaging-service: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	req.AddCookie(cookie)

	client := &http.Client{}
	messagesResp, err := client.Do(req)
	if err != nil {
		log.Printf("[HandleMessages] Failed to fetch messages for user %s: %v", userID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer messagesResp.Body.Close()

	if messagesResp.StatusCode != http.StatusOK {
		log.Printf("[HandleMessages] Error from messaging-service for user %s: %s", userID, messagesResp.Status)
		http.Error(w, "Failed to fetch messages", http.StatusInternalServerError)
		return
	}

	// Decode the response from the messaging service
	var messages []Message
	err = json.NewDecoder(messagesResp.Body).Decode(&messages)
	if err != nil {
		log.Printf("[HandleMessages] Failed to decode messages response for user %s: %v", userID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("Decoded messages is %v", messages[0])

	// Return the messages as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		log.Printf("[HandleMessages] Failed to encode messages response for user %s: %v", userID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
