package api

import (
	"github.com/lai0xn/codek-gateway/pb"
	"github.com/segmentio/kafka-go"
)

type API struct {
	Producer   *kafka.Writer
	RepoClient pb.RepoServiceClient
}
