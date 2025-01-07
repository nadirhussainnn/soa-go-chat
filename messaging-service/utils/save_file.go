// Handle file saving on server
// Author: Nadir Hussain

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// Saves the provided file content to a with a unique name in the server's file system.
//
// Parameters:
// - fileName: string - The original name of the file being uploaded.
// - fileContent: []byte - The binary content of the file to be saved.
//
// Returns:
// - string: The unique file name created for storage (e.g., "example_12345abc.jpg").
// - string: The original file name provided by the user (e.g., "example.jpg").
// - error: Returns an error if the file cannot be saved.

func SaveFile(fileName string, fileContent []byte) (string, string, error) {
	// Define the directory to save files
	directory := "./uploads"

	err := os.MkdirAll(directory, os.ModePerm)
	if err != nil {
		return "", "", fmt.Errorf("failed to create directory: %v", err)
	}

	uniqueID := uuid.New().String()
	// Extract the file extension
	ext := filepath.Ext(fileName)
	baseName := strings.TrimSuffix(fileName, ext)

	// Create a unique filename
	uniqueFileName := fmt.Sprintf("%s_%s%s", baseName, uniqueID, ext)

	// Generate a unique file path
	filePath := filepath.Join(directory, uniqueFileName)
	err = os.WriteFile(filePath, fileContent, 0644)
	if err != nil {
		return "", "", fmt.Errorf("failed to save file: %v", err)
	}

	// Return unique filename (for storage), original filename (for display), and nil error
	return uniqueFileName, fileName, nil
}
