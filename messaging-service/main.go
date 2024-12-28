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

func initDB() *gorm.DB {
	db, err := gorm.Open(sqlite.Open(os.Getenv("DB_NAME")), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to the database: %v", err)
	}

	// Migrate the schema
	db.AutoMigrate(&models.Message{})
	return db
}

func main() {
	// Load environment variables
	utils.LoadEnvs()
	PORT := os.Getenv("PORT")
	AMQP_URL := os.Getenv("AMQP_URL")

	db := initDB()
	repo := repository.NewMessagesRepository(db)

	// Set up RabbitMQ
	conn, ch := amqp.InitRabbitMQ(AMQP_URL)
	defer conn.Close()
	defer ch.Close()

	// Initialize WebSocket handler
	webSocketHandler := utils.NewWebSocketHandler(repo, ch)

	handler := &handlers.MessageHandler{Repo: repo}

	// Use the RabbitMQ channel in middleware for session validation
	authMiddleware := &middleware.AuthMiddleware{
		AMQPChannel: ch, // Correctly pass the RabbitMQ channel

	}

	http.HandleFunc("/ws", webSocketHandler.HandleWebSocket)

	http.Handle("/", authMiddleware.RequireAuth(http.HandlerFunc(handler.GetMessages)))

	log.Println("Messaging service running on port", PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, nil))
}
