// page_renders serves pages, specifically in case of errors/auth
// Author: Nadir Hussain

package utils

import (
	"log"
	"net/http"
	"text/template"
)

// Helper function to render the error page
func RenderErrorPage(w http.ResponseWriter, errorMessage string) {
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	err := tmpl.ExecuteTemplate(w, "error.html", map[string]interface{}{
		"ErrorMessage": errorMessage,
	})
	if err != nil {
		log.Printf("Failed to render error page: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Helper function to render the login template with an error message
func RenderLoginWithError(w http.ResponseWriter, errorMessage string) {
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	err := tmpl.ExecuteTemplate(w, "login.html", map[string]interface{}{
		"ErrorMessage": errorMessage,
	})
	if err != nil {
		log.Printf("[renderLoginWithError] Failed to render login template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Helper function to render the login template with an error message
func RenderRegisterWithError(w http.ResponseWriter, errorMessage string) {
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	err := tmpl.ExecuteTemplate(w, "register.html", map[string]interface{}{
		"ErrorMessage": errorMessage,
	})
	if err != nil {
		log.Printf("[renderRegisterWithError] Failed to render register template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Helper function to render the login template with an error message
func RenderResetPassWithError(w http.ResponseWriter, errorMessage string) {
	tmpl := template.Must(template.ParseGlob("templates/*.html"))
	err := tmpl.ExecuteTemplate(w, "forgot_password.html", map[string]interface{}{
		"ErrorMessage": errorMessage,
	})
	if err != nil {
		log.Printf("[renderResetPasswordWithError] Failed to render reset_password template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}
