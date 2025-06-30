package infra

import (
	"fmt"
	"os"

	"github.com/segmentio/kafka-go"
)

func MakeKafkaProducer() (*kafka.Writer, error) {
	kafkaHost := os.Getenv("KAFKA_HOST")
	if kafkaHost == "" {
		return nil, fmt.Errorf("KAFKA_HOST environment variable is not set")
	}

	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{kafkaHost},
		Topic:    "video-chunks",
		Balancer: &kafka.LeastBytes{},
	})

	return writer, nil
}

