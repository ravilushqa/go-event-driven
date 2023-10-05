package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := watermill.NewStdLogger(false, false)

	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	subscriber, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client: rdb,
	}, logger)
	if err != nil {
		panic(err)
	}

	messages, err := subscriber.Subscribe(ctx, "progress")
	if err != nil {
		panic(err)
	}

	for msg := range messages {
		fmt.Printf("Message ID: %s - %s%%\n", msg.UUID, msg.Payload)
		msg.Ack()
	}
}
