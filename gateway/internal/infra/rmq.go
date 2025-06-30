package infra

import (
	"os"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

var rmqCon *amqp.Connection

func RMQConnect() error {
  godotenv.Load()
  con,err := amqp.Dial(os.Getenv("RMQ_HOST"))
  if err != nil {
     return err
  }
  rmqCon = con
  return nil
} 
