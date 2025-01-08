package main

import (
	"fmt"
	"gateway/middleware"
	"gateway/utils"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found. Using environment variables instead.")
	}

	PORT := os.Getenv("PORT")
	AUTH_SERVICE_URL := os.Getenv("AUTH_SERVICE_URL")
	MESSAGING_SERVICE_URL := os.Getenv("MESSAGING_SERVICE_URL")
	CONTACTS_SERVICE_URL := os.Getenv("CONTACTS_SERVICE_URL")
	CONTACTS_SERVICE_WS_URL := os.Getenv("CONTACTS_SERVICE_WS_URL")
	MESSAGING_SERVICE_WS_URL := os.Getenv("MESSAGING_SERVICE_WS_URL")

	r := mux.NewRouter()

	r.HandleFunc("/auth/{path:.*}", utils.HttpProxyHandler(AUTH_SERVICE_URL, "/auth")).Methods("POST", "GET")
	r.HandleFunc("/contacts/{path:.*}", utils.HttpProxyHandler(CONTACTS_SERVICE_URL, "/contacts")).Methods("POST", "GET", "DELETE")
	r.HandleFunc("/messages/{path:.*}", utils.HttpProxyHandler(MESSAGING_SERVICE_URL, "/messages")).Methods("POST", "GET")

	// WebSocket Proxy
	serviceURLs := map[string]string{
		"contacts": CONTACTS_SERVICE_WS_URL,
		"messages": MESSAGING_SERVICE_WS_URL,
	}

	r.HandleFunc("/ws/{path}", utils.WebSocketProxyHandler(serviceURLs)).Methods("GET", "POST")

	handler := middleware.CorsMiddleware(r)

	log.Println("Gateway running on port", PORT)
	log.Fatal(http.ListenAndServe(fmt.Sprint(":", PORT), handler))
}
