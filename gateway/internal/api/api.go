package api

import (
	"codek7/common/pb"

	"github.com/segmentio/kafka-go"
)

type API struct {
	Producer *kafka.Writer
  repoClient pb.RepoServiceClient
}
