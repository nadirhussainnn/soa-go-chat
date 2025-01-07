// Entry point for the messaing Service application. Initializes the database, RabbitMQ, HTTP and WebSockets
// Author: Nadir Hussain

package main

import (
	"log"
	"messaging-service/amqp"
	"messaging-service/handlers"
	"messaging-service/middleware"
	"messaging-service/models"
	"messaging-service/repository"
	"messaging-service/utils"
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

	// Apply schema migrations for message model.
	db.AutoMigrate(&models.Message{})
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
	handler := &handlers.MessageHandler{
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
	http.Handle("/", authMiddleware.RequireAuth(http.HandlerFunc(handler.FetchMessages)))
	http.Handle("/file/", authMiddleware.RequireAuth(http.HandlerFunc(handler.ServeFile)))

	log.Println("Messaging service running on port", PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, nil))
}
