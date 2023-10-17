package main

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

type NotificationShouldBeSent struct {
	NotificationID string
	Email          string
	Message        string
}

type Sender interface {
	SendNotification(ctx context.Context, notificationID, email, message string) error
}

func NewProcessor(router *message.Router, sender Sender, sub message.Subscriber, watermillLogger watermill.LoggerAdapter) *cqrs.EventProcessor {
	eventProcessor, err := cqrs.NewEventProcessorWithConfig(
		router,
		cqrs.EventProcessorConfig{
			GenerateSubscribeTopic: func(params cqrs.EventProcessorGenerateSubscribeTopicParams) (string, error) {
				return "events", nil
			},
			SubscriberConstructor: func(params cqrs.EventProcessorSubscriberConstructorParams) (message.Subscriber, error) {
				return sub, nil
			},
			Marshaler: cqrs.JSONMarshaler{
				GenerateName: cqrs.StructName,
			},
			Logger: watermillLogger,
		},
	)
	if err != nil {
		panic(err)
	}

	err = eventProcessor.AddHandlers(cqrs.NewEventHandler(
		"send_notification",
		func(ctx context.Context, event *NotificationShouldBeSent) error {
			fmt.Println("Sending notification", event.NotificationID, event.Email, event.Message)
			return sender.SendNotification(ctx, event.NotificationID, event.Email, event.Message)
		},
	))
	if err != nil {
		panic(err)
	}

	return eventProcessor
}
