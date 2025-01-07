package utils

import (
	"log"
	"regexp"
	"strings"
	"unicode"
)

// httpError creates an error message for HTTP responses
func httpError(message string) error {
	return &ValidationError{Message: message}
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// validateRegistrationInput validates the user input for registration
func ValidateRegistrationInput(username, email, password string) error {
	// Validate email
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	if match, _ := regexp.MatchString(emailRegex, email); !match {
		return httpError("Invalid email address")
	}

	// Validate username
	usernameRegex := `^[a-zA-Z][a-zA-Z0-9_-]{2,}$`
	if match, _ := regexp.MatchString(usernameRegex, username); !match {
		return httpError("Invalid username. Must start with a letter and be at least 3 characters.")
	}

	ValidatePassword(password)
	return nil
}

// validateRegistrationInput validates the user input for registration
func ValidatePassword(password string) error {

	// Trim password
	password = strings.TrimSpace(password)

	// Validate password
	if len(password) < 6 {
		return httpError("Password must be atleast 6 characters")
	}

	// Check for required character types in the password
	hasUpper := false
	hasLower := false
	hasNumber := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return httpError("Password must include at least one uppercase letter")
	}
	if !hasLower {
		return httpError("Password must include at least one lowercase letter")
	}
	if !hasNumber {
		return httpError("Password must include at least one number")
	}
	if !hasSpecial {
		return httpError("Password must include at least one special character")
	}

	log.Printf("Password passed validation: '%s'", password)

	return nil
}
