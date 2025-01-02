package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

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
