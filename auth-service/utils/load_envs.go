// Load envs anywhere within app to access them. Loaded once in main.go
// Author: Nadir Hussain

package utils

import (
	"log"

	"github.com/joho/godotenv"
)

// Loads envs from .env file
// Params:
//   - None
//
// Returns:
//   - Nonee
func LoadEnvs() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found. Using environment variables instead.")
	}
}
