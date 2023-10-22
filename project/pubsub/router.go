package pubsub

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"

	"tickets/db/read_model_ops_bookings"
	"tickets/entity"
	"tickets/pubsub/command"
	"tickets/pubsub/event"
	"tickets/pubsub/outbox"
)

type DataLake interface {
	Store(ctx context.Context, id string, header entity.EventHeader, eventName string, payload []byte) error
}

func NewWatermillRouter(
	postgresSubscriber message.Subscriber,
	redisPublisher message.Publisher,
	redisSubscriber message.Subscriber,
	eventProcessorConfig cqrs.EventProcessorConfig,
	eventHandler event.Handler,
	commandProcessorConfig cqrs.CommandProcessorConfig,
	commandsHandler command.Handler,
	opsReadModel read_model_ops_bookings.OpsBookingReadModel,
	dataLake DataLake,
	watermillLogger watermill.LoggerAdapter,
) (*message.Router, error) {
	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return nil, fmt.Errorf("could not create router: %w", err)
	}

	useMiddlewares(router, watermillLogger)

	outbox.AddForwarderHandler(postgresSubscriber, redisPublisher, router, watermillLogger)

	eventProcessor, err := cqrs.NewEventProcessorWithConfig(router, eventProcessorConfig)
	if err != nil {
		return nil, fmt.Errorf("could not create event processor: %w", err)
	}

	err = eventProcessor.AddHandlers(
		eventHandler.StoreTicketHandler(),
		eventHandler.AppendToTrackerHandler(),
		eventHandler.IssueReceiptHandler(),
		eventHandler.CancelTicketHandler(),
		eventHandler.DeleteTicketHandler(),
		eventHandler.PrintTicketHandler(),
		eventHandler.PostTicketBookingHandler(),
		cqrs.NewEventHandler(
			"ops_read_model.OnBookingMade",
			opsReadModel.OnBookingMade,
		),
		cqrs.NewEventHandler(
			"ops_read_model.OnTicketReceiptIssued",
			opsReadModel.OnTicketReceiptIssued,
		),
		cqrs.NewEventHandler(
			"ops_read_model.OnTicketBookingConfirmed",
			opsReadModel.OnTicketBookingConfirmed,
		),
		cqrs.NewEventHandler(
			"ops_read_model.OnTicketPrinted",
			opsReadModel.OnTicketPrinted,
		),
		cqrs.NewEventHandler(
			"ops_read_model.OnTicketRefunded",
			opsReadModel.OnTicketRefunded,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("could not add handlers to event processor: %w", err)
	}

	commandProcessor, err := cqrs.NewCommandProcessorWithConfig(router, commandProcessorConfig)
	if err != nil {
		return nil, fmt.Errorf("could not create command processor: %w", err)
	}

	err = commandProcessor.AddHandlers(
		commandsHandler.RefundTicketHandler(),
	)
	if err != nil {
		return nil, fmt.Errorf("could not add handlers to command processor: %w", err)
	}

	router.AddNoPublisherHandler(
		"events_splitter",
		"events",
		redisSubscriber,
		func(msg *message.Message) error {
			eventName := eventProcessorConfig.Marshaler.NameFromMessage(msg)
			if eventName == "" {
				return fmt.Errorf("could not get event name from message")
			}

			topic := "events." + eventName

			return redisPublisher.Publish(topic, msg)
		},
	)

	router.AddNoPublisherHandler(
		"store_to_data_lake",
		"events",
		redisSubscriber,
		func(msg *message.Message) error {
			eventName := eventProcessorConfig.Marshaler.NameFromMessage(msg)
			if eventName == "" {
				return fmt.Errorf("could not get event name from message")
			}

			// we just need to unmarshal event header, rest is stored as is
			type Event struct {
				Header entity.EventHeader `json:"header"`
			}

			var event Event
			if err := eventProcessorConfig.Marshaler.Unmarshal(msg, &event); err != nil {
				return fmt.Errorf("could not unmarshal event: %w", err)
			}

			return dataLake.Store(
				msg.Context(),
				event.Header.ID,
				event.Header,
				eventName,
				msg.Payload,
			)
		},
	)

	return router, nil
}
