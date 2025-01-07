package tests

import (
	"auth-service/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTestDB initializes a shared in-memory SQLite database for testing.
func setupTestDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // Suppress logs during tests
	})
	if err != nil {
		panic("Failed to initialize test database")
	}

	db.AutoMigrate(&models.User{}, &models.Session{})
	return db
}
