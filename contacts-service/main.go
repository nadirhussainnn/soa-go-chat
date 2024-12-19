package main

import (
	"auth-service/handlers"
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
	db.AutoMigrate(&models.User{})
	return db
}

func main() {

	utils.LoadEnvs()

	PORT = os.Getenv("PORT")
	DB_NAME = os.Getenv("DB_NAME")

	db := initDB()
	userRepo := repository.NewUserRepository(db)

	handler := &handlers.Handler{UserRepo: userRepo}

	http.HandleFunc("/register", handler.RegisterHandler)
	http.HandleFunc("/login", handler.LoginHandler)
	http.HandleFunc("/forgot-password", handler.ForgotPasswordHandler)

	// // Example protected route
	// http.Handle("/protected", middleware.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	// 	w.Write([]byte("This is a protected route"))
	// })))

	log.Println("Auth service running on port", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprint(":", PORT), nil))
}
