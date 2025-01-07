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

func initDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(DB_NAME), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	// Migrate the schema
	db.AutoMigrate(&models.User{}, &models.Session{})
	return db
}

func main() {

	utils.LoadEnvs()

	PORT = os.Getenv("PORT")
	DB_NAME = os.Getenv("DB_NAME")
	AMQP_URL := os.Getenv("AMQP_URL")

	db := initDB()
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)

	// Set up RabbitMQ
	conn, ch := amqp.InitRabbitMQ(AMQP_URL)
	defer conn.Close()
	defer ch.Close()

	sessionVerifier := amqp.SessionVerifier{SessionRepo: sessionRepo}
	sessionVerifier.ListenForSessionVerification(ch)

	jwtDecoder := amqp.JWTDecoder{Secret: os.Getenv("JWT_SECRET")}
	jwtDecoder.ListenForJWTDecode(conn)

	amqp.ListenForBatchDetails(ch, userRepo)

	handler := &handlers.Handler{UserRepo: userRepo, SessionRepo: sessionRepo}

	http.HandleFunc("/register", handler.RegisterHandler)
	http.HandleFunc("/login", handler.LoginHandler)
	http.HandleFunc("/forgot-password", handler.ForgotPasswordHandler)
	http.Handle("/logout", middleware.RequireAuth(http.HandlerFunc(handler.LogoutHandler)))
	// http.HandleFunc("/logout", handler.LogoutHandler)
	http.Handle("/search", middleware.RequireAuth(http.HandlerFunc(handler.SearchContacts)))

	log.Println("Auth service running on port", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprint(":", PORT), nil))
}
