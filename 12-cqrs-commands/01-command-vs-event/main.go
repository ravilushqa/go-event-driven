package main

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
)

type SendNotification struct {
	NotificationID string
	Email          string
	Message        string
}

type Sender interface {
	SendNotification(ctx context.Context, notificationID, email, message string) error
}

func NewProcessor(router *message.Router, sender Sender, sub message.Subscriber, watermillLogger watermill.LoggerAdapter) *cqrs.CommandProcessor {
	commandProcessor, err := cqrs.NewCommandProcessorWithConfig(
		router,
		cqrs.CommandProcessorConfig{
			GenerateSubscribeTopic: func(params cqrs.CommandProcessorGenerateSubscribeTopicParams) (string, error) {
				return params.CommandName, nil
			},
			SubscriberConstructor: func(params cqrs.CommandProcessorSubscriberConstructorParams) (message.Subscriber, error) {
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

	err = commandProcessor.AddHandlers(cqrs.NewCommandHandler(
		"send_notification",
		func(ctx context.Context, event *SendNotification) error {
			fmt.Println("Sending notification", event.NotificationID, event.Email, event.Message)
			return sender.SendNotification(ctx, event.NotificationID, event.Email, event.Message)
		},
	))
	if err != nil {
		panic(err)
	}

	return commandProcessor
}
