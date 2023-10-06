package pubsub

import (
	"encoding/json"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/redis/go-redis/v9"

	"tickets/entity"
)

const brokenMessageID = "2beaf5bc-d5e4-4653-b075-2b36bbf28949"

func NewWatermillRouter(receiptsService ReceiptsService, spreadsheetsService SpreadsheetsAPI, rdb *redis.Client, watermillLogger watermill.LoggerAdapter) *message.Router {
	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		panic(err)
	}

	handler := NewHandler(spreadsheetsService, receiptsService)

	useMiddlewares(router, watermillLogger)

	issueReceiptSub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        rdb,
		ConsumerGroup: "issue-receipt",
	}, watermillLogger)
	if err != nil {
		panic(err)
	}

	appendToTrackerSub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        rdb,
		ConsumerGroup: "append-to-tracker",
	}, watermillLogger)
	if err != nil {
		panic(err)
	}

	cancelTicketSub, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        rdb,
		ConsumerGroup: "cancel-ticket",
	}, watermillLogger)
	if err != nil {
		panic(err)
	}

	router.AddNoPublisherHandler(
		"issue_receipt",
		"TicketBookingConfirmed",
		issueReceiptSub,
		func(msg *message.Message) error {
			// Fixing a malformed JSON message
			if string(msg.UUID) == brokenMessageID {
				return nil
			}

			// Fixing an incorrect message type
			if msg.Metadata.Get("type") != "TicketBookingConfirmed" {
				return nil
			}

			var event entity.TicketBookingConfirmed
			err := json.Unmarshal(msg.Payload, &event)
			if err != nil {
				return err
			}

			// Fixing a code bug: for some events, we didn't supply the currency, which was USD by default
			// Now some events are spinning
			// Add this if to default to USD for these events
			if event.Price.Currency == "" {
				event.Price.Currency = "USD"
			}

			return handler.IssueReceipt(msg.Context(), event)
		},
	)

	router.AddNoPublisherHandler(
		"append_to_tracker",
		"TicketBookingConfirmed",
		appendToTrackerSub,
		func(msg *message.Message) error {
			// Fixing a malformed JSON message
			if string(msg.UUID) == brokenMessageID {
				return nil
			}

			// Fixing an incorrect message type
			if msg.Metadata.Get("type") != "TicketBookingConfirmed" {
				return nil
			}

			var event entity.TicketBookingConfirmed
			err := json.Unmarshal(msg.Payload, &event)
			if err != nil {
				return err
			}

			// Fixing a code bug: for some events, we didn't supply the currency, which was USD by default
			// Now some events are spinning
			// Add this if to default to USD for these events
			if event.Price.Currency == "" {
				event.Price.Currency = "USD"
			}

			return handler.AppendToTracker(msg.Context(), event)
		},
	)

	router.AddNoPublisherHandler(
		"cancel_ticket",
		"TicketBookingCanceled",
		cancelTicketSub,
		func(msg *message.Message) error {
			// Fixing an incorrect message type
			if msg.Metadata.Get("type") != "TicketBookingCanceled" {
				return nil
			}

			var event entity.TicketBookingCanceled
			err := json.Unmarshal(msg.Payload, &event)
			if err != nil {
				return err
			}
			return handler.CancelTicket(msg.Context(), event)
		},
	)

	return router
}
