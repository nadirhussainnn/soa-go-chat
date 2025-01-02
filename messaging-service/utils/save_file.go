package utils

import (
	"fmt"
	"os"
)

func SaveFile(fileName string, fileContent []byte) (string, error) {
	// Define the directory to save files
	directory := "./uploads"
	err := os.MkdirAll(directory, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to create directory: %v", err)
	}

	// Generate a unique file path
	filePath := fmt.Sprintf("%s/%s", directory, fileName)
	err = os.WriteFile(filePath, fileContent, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to save file: %v", err)
	}

	return filePath, nil
}
