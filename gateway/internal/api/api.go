package api

import (
	"codek7/common/pb"

	"github.com/lumbrjx/codek7/gateway/internal/watcher"
	"github.com/segmentio/kafka-go"
)

type API struct {
	Producer   *kafka.Writer
	RepoClient pb.RepoServiceClient
	Hub        *watcher.Hub
}
