package main

import (
	"contacts-service/amqp"
	"contacts-service/handlers"
	"contacts-service/middleware"
	"contacts-service/models"
	"contacts-service/repository"
	"contacts-service/utils"
	"log"
	"net/http"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func initDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(os.Getenv("DB_NAME")), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	// Migrate the schema
	db.AutoMigrate(&models.Contact{}, &models.ContactRequest{})
	return db
}

func main() {
	// Load environment variables
	utils.LoadEnvs()
	PORT := os.Getenv("PORT")
	AMQP_URL := os.Getenv("AMQP_URL")

	db := initDB()
	repo := repository.NewContactsRepository(db)

	// Set up RabbitMQ
	conn, ch := amqp.InitRabbitMQ(AMQP_URL)
	defer conn.Close()
	defer ch.Close()

	// Initialize WebSocket handler
	webSocketHandler := utils.NewWebSocketHandler(repo, ch)

	handler := &handlers.ContactsHandler{Repo: repo}

	authMiddleware := &middleware.AuthMiddleware{
		AMQPConn: conn, // Pass the RabbitMQ connection
	}

	http.HandleFunc("/ws", webSocketHandler.HandleWebSocket)

	http.Handle("/", authMiddleware.RequireAuth(http.HandlerFunc(handler.GetContacts)))

	log.Println("Contacts service running on port", PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, nil))
}
