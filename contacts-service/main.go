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
	conn, _ := amqp.InitRabbitMQ(AMQP_URL) // Connection setup
	defer conn.Close()

	// Initialize WebSocket handler
	webSocketHandler := utils.NewWebSocketHandler(repo, nil) // Pass nil as channel is now dynamic

	handler := &handlers.ContactsHandler{
		Repo:             repo,
		WebSocketHandler: webSocketHandler,
		AMQPConn:         conn, // Pass the RabbitMQ connection
	}

	authMiddleware := &middleware.AuthMiddleware{
		AMQPConn: conn,
	}

	http.HandleFunc("/ws", webSocketHandler.HandleWebSocket)

	http.Handle("/", authMiddleware.RequireAuth(http.HandlerFunc(handler.GetContacts)))
	http.Handle("/requests/", authMiddleware.RequireAuth(http.HandlerFunc(handler.FetchPendingRequests)))

	http.HandleFunc("/contacts/request/action", handlers.HandleRequestAction)
	log.Println("Contacts service running on port", PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, nil))
}
