// Provides RabbitMQ connection and channel initialization.
// Author: Nadir Hussain

package amqp

import (
	"log"

	"github.com/streadway/amqp"
)

// Initializes a RabbitMQ connection and channel.

// Parameters:
// - amqpURL: RabbitMQ connection URL.

// Returns:
// - *amqp.Connection: The RabbitMQ connection.
// - *amqp.Channel: The opened channel.
func InitRabbitMQ(amqpURL string) (*amqp.Connection, *amqp.Channel) {

	// Connecting to RabbitMQ
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	// Opening a channel
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	return conn, ch
}
