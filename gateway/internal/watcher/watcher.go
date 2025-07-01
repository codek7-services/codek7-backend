package watcher

import (
	"encoding/json"
	"log"
	"time"

	"github.com/lumbrjx/codek7/gateway/internal/infra"
	amqp "github.com/rabbitmq/amqp091-go"
)

// Watcher handles RabbitMQ message consumption and WebSocket broadcasting
type Watcher struct {
	hub        *Hub
	connection *amqp.Connection
	channel    *amqp.Channel
	queueName  string
}

// NewWatcher creates a new Watcher instance
func NewWatcher(hub *Hub) (*Watcher, error) {
	// Connect to RabbitMQ
	if err := infra.RMQConnect(); err != nil {
		return nil, err
	}

	conn := infra.GetRMQConnection()
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		hub:        hub,
		connection: conn,
		channel:    ch,
		queueName:  "notify.q",
	}, nil
}

// Start begins consuming messages from RabbitMQ
func (w *Watcher) Start() error {
	// Declare the queue
	_, err := w.channel.QueueDeclare(
		w.queueName, // name
		true,        // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	if err != nil {
		return err
	}

	// Set QoS to limit unacknowledged messages
	err = w.channel.Qos(
		10,    // prefetch count
		0,     // prefetch size
		false, // global
	)
	if err != nil {
		return err
	}

	// Start consuming messages
	msgs, err := w.channel.Consume(
		w.queueName, // queue
		"",          // consumer
		false,       // auto-ack
		false,       // exclusive
		false,       // no-local
		false,       // no-wait
		nil,         // args
	)
	if err != nil {
		return err
	}

	log.Printf("Watcher started, consuming from queue: %s", w.queueName)

	// Process messages in a goroutine
	go func() {
		for msg := range msgs {
			w.processMessage(msg)
		}
	}()

	return nil
}

func (w *Watcher) processMessage(msg amqp.Delivery) {
	var notification Notification

	// Parse the JSON message
	if err := json.Unmarshal(msg.Body, &notification); err != nil {
		log.Printf("Error parsing notification: %v", err)
		msg.Nack(false, false) // Reject the message
		return
	}

	// Add timestamp if not present
	if notification.Timestamp.IsZero() {
		notification.Timestamp = time.Now()
	}

	log.Printf("Received notification for user %s: %s from %s",
		notification.UserID, notification.EventType, notification.ServiceName)

	// Send to WebSocket hub
	w.hub.SendNotification(notification)

	// Acknowledge the message
	msg.Ack(false)
}

// Close closes the RabbitMQ connection
func (w *Watcher) Close() error {
	if w.channel != nil {
		w.channel.Close()
	}
	if w.connection != nil {
		return w.connection.Close()
	}
	return nil
}
