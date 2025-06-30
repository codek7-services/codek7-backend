package api

import (
	"github.com/segmentio/kafka-go"
)

type API struct {
	Producer *kafka.Writer
}
