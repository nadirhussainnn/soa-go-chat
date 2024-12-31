package amqp

import (
	"consumer/utils"
	"log"

	"github.com/streadway/amqp"
)

type AMQPConfig struct {
	Connection *amqp.Connection
	Channel    *amqp.Channel
}

// NewAMQPConfig initializes the RabbitMQ connection and channel
func NewAMQPConfig(amqpURL string) (*AMQPConfig, error) {
	// Connect to RabbitMQ
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, err
	}

	// Open a channel
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	// Declare queues
	err = declareQueues(ch)
	if err != nil {
		conn.Close()
		ch.Close()
		return nil, err
	}

	return &AMQPConfig{Connection: conn, Channel: ch}, nil
}

// declareQueues declares the necessary queues
func declareQueues(ch *amqp.Channel) error {
	// Declare the session verification queue
	_, err := ch.QueueDeclare(
		utils.AUTH_SESSION_VERIFICATION, // Queue name
		false,                           // Durable
		false,                           // Delete when unused
		false,                           // Exclusive
		false,                           // No-wait
		nil,                             // Arguments
	)
	if err != nil {
		return err
	}

	// Declare the session response queue
	_, err = ch.QueueDeclare(
		utils.AUTH_SESSION_RESPONSE, // Queue name
		false,                       // Durable
		false,                       // Delete when unused
		false,                       // Exclusive
		false,                       // No-wait
		nil,                         // Arguments
	)
	return err
}

// Close cleans up the RabbitMQ connection and channel
func (c *AMQPConfig) Close() {
	if c.Channel != nil {
		c.Channel.Close()
	}
	if c.Connection != nil {
		c.Connection.Close()
	}
}

// InitRabbitMQ sets up the RabbitMQ connection and channel
func InitRabbitMQ(amqpURL string) (*amqp.Connection, *amqp.Channel) {
	// Connect to RabbitMQ
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}

	// Open a channel
	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}

	return conn, ch
}
