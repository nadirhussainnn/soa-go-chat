// Entry point for the Auth Service application. Initializes the database, RabbitMQ, and HTTP routes.
// Author: Nadir Hussain

package main

import (
	"auth-service/amqp"
	"auth-service/handlers"
	"auth-service/middleware"
	"auth-service/models"
	"auth-service/repository"
	"auth-service/utils"
	"fmt"
	"log"
	"net/http"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var PORT, DB_NAME string

// Initializes the database connection and applies migrations.
// Returns:
//   - *gorm.DB: Database instance for further operations.
func initDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(DB_NAME), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	// Apply schema migrations for User and Session models.
	db.AutoMigrate(&models.User{}, &models.Session{})
	return db
}

func main() {

	// Loading environment variables from the .env file.
	utils.LoadEnvs()

	// Setting configuration variables.
	PORT = os.Getenv("PORT")
	DB_NAME = os.Getenv("DB_NAME")
	AMQP_URL := os.Getenv("AMQP_URL")

	// Initializing database and repositories.
	db := initDB()
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	// Initializing RabbitMQ connection and channel.
	conn, ch := amqp.InitRabbitMQ(AMQP_URL)
	defer conn.Close()
	defer ch.Close()

	// Setting up RabbitMQ consumers.
	sessionVerifier := amqp.SessionVerifier{SessionRepo: sessionRepo}
	sessionVerifier.ListenForSessionVerification(ch)

	jwtDecoder := amqp.JWTDecoder{Secret: os.Getenv("JWT_SECRET")}
	jwtDecoder.ListenForJWTDecode(conn)

	amqp.ListenForBatchDetails(ch, userRepo)

	// Initializing HTTP handlers.
	handler := &handlers.Handler{UserRepo: userRepo, SessionRepo: sessionRepo}

	// Define HTTP routes with middleware for authentication where needed.
	http.HandleFunc("/register", handler.RegisterHandler)
	http.HandleFunc("/login", handler.LoginHandler)
	http.HandleFunc("/forgot-password", handler.ForgotPasswordHandler)

	// Protected routes i.e need to be logged in to use these APIs
	http.Handle("/logout", middleware.RequireAuth(http.HandlerFunc(handler.LogoutHandler)))
	http.Handle("/search", middleware.RequireAuth(http.HandlerFunc(handler.SearchContacts)))

	log.Println("Auth service running on port", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprint(":", PORT), nil))
}
