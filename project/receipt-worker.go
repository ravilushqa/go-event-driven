package main

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

type IssueWorker struct {
	topic          string
	pub            message.Publisher
	sub            message.Subscriber
	receiptsClient ReceiptsClient
}

func NewIssueWorker(pub message.Publisher, sub message.Subscriber, receiptsClient ReceiptsClient) *IssueWorker {
	return &IssueWorker{
		topic:          "issue-receipt",
		pub:            pub,
		sub:            sub,
		receiptsClient: receiptsClient,
	}
}

func (w *IssueWorker) Run(ctx context.Context) error {
	messages, err := w.sub.Subscribe(ctx, w.topic)
	if err != nil {
		return err
	}

	for msg := range messages {
		err = w.receiptsClient.IssueReceipt(msg.Context(), string(msg.Payload))
		if err != nil {
			msg.Nack()
		} else {
			msg.Ack()
		}
	}

	return nil
}

func (w *IssueWorker) Send(msg ...*message.Message) error {
	for _, m := range msg {
		err := w.pub.Publish(w.topic, m)
		if err != nil {
			return err
		}
	}

	return nil
}
