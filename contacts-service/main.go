// Entry point for the contacts Service application. Initializes the database, RabbitMQ, HTTP and WebSockets
// Author: Nadir Hussain

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

// Initializes the database connection and applies migrations.
// Returns:
//   - *gorm.DB: Database instance for further operations.

func initDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(os.Getenv("DB_NAME")), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	// Apply schema migrations for contacts and ContactRequests models.
	db.AutoMigrate(&models.Contact{}, &models.ContactRequest{})
	return db
}

func main() {

	// Loading environment variables from the .env file.
	utils.LoadEnvs()

	// Setting configuration variables.
	PORT := os.Getenv("PORT")
	AMQP_URL := os.Getenv("AMQP_URL")

	// Initializing database and repositories.
	db := initDB()
	repo := repository.NewContactsRepository(db)

	// Initializing RabbitMQ connection and channel.
	conn, ch := amqp.InitRabbitMQ(AMQP_URL)
	defer conn.Close()
	defer ch.Close()

	// Initialize WebSocket handler
	webSocketHandler := utils.NewWebSocketHandler(repo, ch)

	// Initializing handler with repo, websocket and amqp conn
	handler := &handlers.ContactsHandler{
		Repo:             repo,
		WebSocketHandler: webSocketHandler,
		AMQPConn:         conn, // Pass the RabbitMQ connection
	}

	// Initializing middleware
	authMiddleware := &middleware.AuthMiddleware{
		AMQPConn: conn,
	}

	// Listen for web socket connections - handle all web socket envts in HandleWebSocket
	http.HandleFunc("/ws", webSocketHandler.HandleWebSocket)

	// Handle REST APIs
	http.Handle("/", authMiddleware.RequireAuth(http.HandlerFunc(handler.GetContacts)))
	http.Handle("/requests/", authMiddleware.RequireAuth(http.HandlerFunc(handler.FetchPendingRequests)))

	log.Println("Contacts service running on port", PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, nil))
}
