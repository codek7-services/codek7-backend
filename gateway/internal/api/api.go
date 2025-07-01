package api

import (
	"github.com/lumbrjx/codek7/gateway/pb"
	"github.com/segmentio/kafka-go"
)

type API struct {
	Producer   *kafka.Writer
	RepoClient pb.RepoServiceClient
}
