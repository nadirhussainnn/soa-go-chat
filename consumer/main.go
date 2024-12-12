package main

import (
	"log"

	"github.com/streadway/amqp"
)

func StartConsumer() {
	// Connect to RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	// Open a channel
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	// Declare the queue
	q, err := ch.QueueDeclare(
		"messages", // Queue name
		false,      // Durable
		false,      // Delete when unused
		false,      // Exclusive
		false,      // No-wait
		nil,        // Arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %v", err)
	}

	// Consume messages
	msgs, err := ch.Consume(
		q.Name, // Queue name
		"",     // Consumer
		true,   // Auto-acknowledge
		false,  // Exclusive
		false,  // No-local
		false,  // No-wait
		nil,    // Args
	)
	if err != nil {
		log.Fatalf("Failed to register a consumer: %v", err)
	}

	forever := make(chan bool)

	// Process messages in a goroutine
	go func() {
		for d := range msgs {
			log.Printf("Received message: %s", d.Body)
			// Add your processing logic here
		}
	}()

	log.Println("Waiting for messages. To exit press CTRL+C")
	<-forever
}

func main() {
	log.Println("Message Consumer Service started")
	StartConsumer() // Start the consumer
}
