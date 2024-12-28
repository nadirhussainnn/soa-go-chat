package auth

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"os"
)

func HandleDashboard(w http.ResponseWriter, r *http.Request) {
	GATEWAY_URL := os.Getenv("GATEWAY_URL")

	cookie, err := r.Cookie("session_token")
	if err != nil || cookie.Value == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userID := r.Context().Value("user_id").(string)
	req, err := http.NewRequest("GET", GATEWAY_URL+"/contacts/?user_id="+userID, nil)
	if err != nil {
		log.Printf("Failed to create request to contacts-service: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	req.AddCookie(cookie)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to fetch dashboard data: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error from contacts-service: %s", resp.Status)
		http.Error(w, "Failed to fetch dashboard data", http.StatusInternalServerError)
		return
	}

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
			CreatedAtFormatted string `json:"createdAtFormatted"`
			SenderDetails      struct {
				Username string `json:"username"`
				Email    string `json:"email"`
			} `json:"senderDetails"`
		} `json:"contactRequests"`
	}
	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil {
		log.Printf("Failed to decode dashboard data: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	err = tmpl.ExecuteTemplate(w, "dashboard.html", data)
	if err != nil {
		log.Printf("Failed to render template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
