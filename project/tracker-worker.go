package main

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

type TrackerWorker struct {
	topic              string
	pub                message.Publisher
	sub                message.Subscriber
	spreadsheetsClient SpreadsheetsClient
}

func NewTrackerWorker(pub message.Publisher, sub message.Subscriber, spreadsheetsClient SpreadsheetsClient) *TrackerWorker {
	return &TrackerWorker{
		topic:              "append-to-tracker",
		pub:                pub,
		sub:                sub,
		spreadsheetsClient: spreadsheetsClient,
	}
}

func (w *TrackerWorker) Run(ctx context.Context) error {
	messages, err := w.sub.Subscribe(ctx, w.topic)
	if err != nil {
		return err
	}

	for msg := range messages {
		err = w.spreadsheetsClient.AppendRow(msg.Context(), "tickets-to-print", []string{string(msg.Payload)})
		if err != nil {
			msg.Nack()
		} else {
			msg.Ack()
		}
	}

	return nil

}

func (w *TrackerWorker) Send(msg ...*message.Message) error {
	for _, m := range msg {
		err := w.pub.Publish(w.topic, m)
		if err != nil {
			return err
		}
	}

	return nil
}
