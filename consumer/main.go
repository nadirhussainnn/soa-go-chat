package main

import (
	"consumer/amqp"

	handlers "consumer/handlers"
	"consumer/middleware"
	"consumer/utils"
	"html/template"
	"log"
	"net/http"
	"os"
)

func main() {

	utils.LoadEnvs()

	PORT := os.Getenv("PORT")
	AMQP_URL := os.Getenv("AMQP_URL")

	// Set up RabbitMQ
	conn, _ := amqp.InitRabbitMQ(AMQP_URL) // Connection setup
	defer conn.Close()

	authMiddleware := &middleware.AuthMiddleware{
		AMQPConn: conn,
	}

	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	tmpl := template.Must(template.ParseGlob("./templates/*.html"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "index.html", nil)
	})

	http.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tmpl.ExecuteTemplate(w, "login.html", nil)
		} else if r.Method == http.MethodPost {
			handlers.HandleLogin(w, r)
		}
	})

	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tmpl.ExecuteTemplate(w, "register.html", nil)
		} else if r.Method == http.MethodPost {
			handlers.HandleRegister(w, r)
		}
	})

	http.HandleFunc("/forgot-password", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			tmpl.ExecuteTemplate(w, "forgot_password.html", nil)
		} else {
			handlers.HandleForgotPassword(w, r)
		}
	})

	http.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		handlers.HandleLogout(w, r)
	})

	http.HandleFunc("/terms", func(w http.ResponseWriter, r *http.Request) {
		tmpl.ExecuteTemplate(w, "terms.html", nil)
	})

	http.Handle("/dashboard", authMiddleware.RequireAuth(http.HandlerFunc(handlers.HandleDashboard)))
	http.Handle("/contacts", authMiddleware.RequireAuth(http.HandlerFunc(handlers.HandleContacts)))
	http.Handle("/requests", authMiddleware.RequireAuth(http.HandlerFunc(handlers.HandleRequests)))
	http.Handle("/search", authMiddleware.RequireAuth(http.HandlerFunc(handlers.HandleSearch)))

	http.Handle("/messages", authMiddleware.RequireAuth(http.HandlerFunc(handlers.HandleMessages)))

	// Start the HTTP server
	log.Println("Consumer service running on port", PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, nil))
}
