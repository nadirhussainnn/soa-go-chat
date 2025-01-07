// Performs validation on username, email and password and returns respective error
// Author: Nadir Hussain

package utils

import (
	"log"
	"regexp"
	"strings"
	"unicode"
)

// ValidationError represents a validation error
type ValidationError struct {
	Message string
}

// httpError creates an error message for HTTP responses.
// Params:
//   - message (string): The error message to return.
//
// Returns:
//   - error: A ValidationError with the provided message.
func httpError(message string) error {
	return &ValidationError{Message: message}
}

// Returns the error message for ValidationError.
// Returns:
//   - string: The error message.
func (e *ValidationError) Error() string {
	return e.Message
}

// Validates the user input for registration.
// Validates the email, username, and password fields.
// Params:
//   - username (string): The username provided by the user.
//   - email (string): The email address provided by the user.
//   - password (string): The password provided by the user.
//
// Returns:
//   - error: An error if validation fails, otherwise nil.
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

	// Validate password
	if err := ValidatePassword(password); err != nil {
		return err
	}
	return nil
}

// Validates a password based on complexity requirements.
// The password must include uppercase, lowercase, numeric, and special characters.
// Params:
//   - password (string): The password to validate.
//
// Returns:
//   - error: An error if the password does not meet the requirements, otherwise nil.
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
