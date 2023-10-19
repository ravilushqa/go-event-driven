package pubsub

import (
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"
)

func NewRedisPublisher(rdb *redis.Client, watermillLogger watermill.LoggerAdapter) message.Publisher {
	var publisher message.Publisher
	publisher, err := redisstream.NewPublisher(redisstream.PublisherConfig{
		Client: rdb,
	}, watermillLogger)
	if err != nil {
		panic(err)
	}

	publisher = log.CorrelationPublisherDecorator{Publisher: publisher}
	return publisher
}
