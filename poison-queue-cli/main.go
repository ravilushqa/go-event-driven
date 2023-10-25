package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Shopify/sarama"
	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-kafka/v2/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/urfave/cli/v2"
)

const PoisonQueueTopic = "PoisonQueue"

type Message struct {
	ID     string
	Reason string
}

type Handler struct {
	subscriber message.Subscriber
	publisher  message.Publisher
}

func NewHandler() (*Handler, error) {
	logger := watermill.NewStdLogger(false, false)

	cfg := sarama.NewConfig()
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	sub, err := kafka.NewSubscriber(
		kafka.SubscriberConfig{
			Brokers:               []string{os.Getenv("KAFKA_ADDR")},
			Unmarshaler:           kafka.DefaultMarshaler{},
			ConsumerGroup:         "poison-queue-cli",
			OverwriteSaramaConfig: cfg,
		},
		logger,
	)
	if err != nil {
		return nil, err
	}

	pub, err := kafka.NewPublisher(
		kafka.PublisherConfig{
			Brokers:   []string{os.Getenv("KAFKA_ADDR")},
			Marshaler: kafka.DefaultMarshaler{},
		},
		logger,
	)
	if err != nil {
		return nil, err
	}

	return &Handler{
		subscriber: sub,
		publisher:  pub,
	}, nil
}

func (h *Handler) Preview(ctx context.Context) ([]Message, error) {
	messages, err := h.subscriber.Subscribe(ctx, PoisonQueueTopic)
	if err != nil {
		return nil, err
	}
	defer h.subscriber.Close()

	firstID := ""
	var result []Message

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		for msg := range messages {
			if firstID == msg.UUID {
				fmt.Println("No more messages")
				return result, nil
			}
			fmt.Println(msg.UUID)
			if firstID == "" {
				firstID = msg.UUID
			}

			result = append(result, Message{
				ID:     msg.UUID,
				Reason: msg.Metadata.Get(middleware.ReasonForPoisonedKey),
			})

			err = h.publisher.Publish(PoisonQueueTopic, msg)
			if err != nil {
				return nil, err
			}

			msg.Ack()
		}
	}

	return result, nil
}

func (h *Handler) Remove(ctx context.Context, messageID string) error {
	messages, err := h.subscriber.Subscribe(ctx, PoisonQueueTopic)
	if err != nil {
		return err
	}
	defer h.subscriber.Close()

	firstID := ""

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		for msg := range messages {
			if firstID == msg.UUID {
				return fmt.Errorf("message %s not found", messageID)
			}
			fmt.Println(msg.UUID)
			if firstID == "" {
				firstID = msg.UUID
			}

			if msg.UUID == messageID {
				msg.Ack()
				return nil
			}

			err = h.publisher.Publish(PoisonQueueTopic, msg)
			if err != nil {
				return err
			}

			msg.Ack()
		}
	}

	return nil
}

func main() {
	app := &cli.App{
		Name:  "poison-queue-cli",
		Usage: "Manage the Poison Queue",
		Commands: []*cli.Command{
			{
				Name:  "preview",
				Usage: "preview messages",
				Action: func(c *cli.Context) error {
					h, err := NewHandler()
					if err != nil {
						return err
					}

					messages, err := h.Preview(c.Context)
					if err != nil {
						return err
					}

					for _, m := range messages {
						fmt.Printf("%v\t%v\n", m.ID, m.Reason)
					}

					return nil
				},
			},
			{
				Name:      "remove",
				ArgsUsage: "<message_id>",
				Usage:     "remove message",
				Action: func(c *cli.Context) error {
					h, err := NewHandler()
					if err != nil {
						return err
					}

					err = h.Remove(c.Context, c.Args().First())
					if err != nil {
						return err
					}

					return nil
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
