package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"text/template"
)

// HandleDashboard processes requests to fetch and display the dashboard
func HandleDashboard(w http.ResponseWriter, r *http.Request) {
	// Extract user_id from the request context
	userID, ok := r.Context().Value("user_id").(string)
	if !ok || userID == "" {
		log.Print("User ID not found in context")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	GATEWAY_URL := os.Getenv("GATEWAY_URL")
	// Fetch contacts from contacts-service
	resp, err := http.Get(GATEWAY_URL + "/contacts?user_id=" + userID)
	if err != nil {
		log.Printf("Failed to fetch contacts from contacts-service: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error from contacts-service: %s", resp.Status)
		http.Error(w, "Failed to fetch contacts", http.StatusInternalServerError)
		return
	}

	var contacts []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	err = json.NewDecoder(resp.Body).Decode(&contacts)
	if err != nil {
		log.Printf("Failed to decode contacts response: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Pass contacts to the template for rendering
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	if err := tmpl.ExecuteTemplate(w, "dashboard.html", contacts); err != nil {
		log.Printf("Failed to render template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
