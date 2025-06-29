package infra

import (
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

var rmqCon *amqp.Connection

func RMQConnect() error {
  con,err := amqp.Dial(os.Getenv("RMQ_HOST"))
  if err != nil {
     return err
  }
  rmqCon = con
  return nil
} 
