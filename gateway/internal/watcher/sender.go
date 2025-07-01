package watcher

import (
	"encoding/json"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// NotificationSender provides a utility for sending notifications to RabbitMQ
type NotificationSender struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	queueName  string
}

// NewNotificationSender creates a new notification sender
func NewNotificationSender(rmqURL string) (*NotificationSender, error) {
	conn, err := amqp.Dial(rmqURL)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, err
	}

	queueName := "notify.q"

	// Declare the queue
	_, err = ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, err
	}

	return &NotificationSender{
		connection: conn,
		channel:    ch,
		queueName:  queueName,
	}, nil
}

// SendNotification sends a notification to the queue
func (ns *NotificationSender) SendNotification(notification Notification) error {
	// Set timestamp if not provided
	if notification.Timestamp.IsZero() {
		notification.Timestamp = time.Now()
	}

	// Convert to JSON
	body, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	// Publish to queue
	return ns.channel.Publish(
		"",           // exchange
		ns.queueName, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
}

// Close closes the connection
func (ns *NotificationSender) Close() error {
	if ns.channel != nil {
		ns.channel.Close()
	}
	if ns.connection != nil {
		return ns.connection.Close()
	}
	return nil
}

// Helper functions for common notification types

// SendSuccessNotification sends a success notification
func (ns *NotificationSender) SendSuccessNotification(userID, videoID, serviceName, description string) error {
	return ns.SendNotification(Notification{
		UserID:      userID,
		EventType:   "success",
		VideoID:     videoID,
		ServiceName: serviceName,
		Description: description,
	})
}

// SendErrorNotification sends an error notification
func (ns *NotificationSender) SendErrorNotification(userID, videoID, serviceName, description string) error {
	return ns.SendNotification(Notification{
		UserID:      userID,
		EventType:   "error",
		VideoID:     videoID,
		ServiceName: serviceName,
		Description: description,
	})
}

// SendProgressNotification sends a progress notification
func (ns *NotificationSender) SendProgressNotification(userID, videoID, serviceName, description string) error {
	return ns.SendNotification(Notification{
		UserID:      userID,
		EventType:   "progress",
		VideoID:     videoID,
		ServiceName: serviceName,
		Description: description,
	})
}
