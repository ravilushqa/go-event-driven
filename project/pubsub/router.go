package pubsub

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"

	"tickets/entity"
	"tickets/pubsub/command"
	"tickets/pubsub/event"
	"tickets/pubsub/outbox"
)

type DataLake interface {
	StoreEvent(ctx context.Context, dataLakeEvent entity.DataLakeEvent) error
}

func NewWatermillRouter(
	postgresSubscriber message.Subscriber,
	redisPublisher message.Publisher,
	redisSubscriber message.Subscriber,
	eventProcessorConfig cqrs.EventProcessorConfig,
	eventHandler event.Handler,
	commandProcessorConfig cqrs.CommandProcessorConfig,
	commandsHandler command.Handler,
	opsBookingsHandlers event.OpsBookingHandlers,
	dataLake DataLake,
	vipBundleProcessManager *entity.VipBundleProcessManager,
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
			opsBookingsHandlers.OnBookingMade,
		),
		cqrs.NewEventHandler(
			"ops_read_model.OnTicketReceiptIssued",
			opsBookingsHandlers.OnTicketReceiptIssued,
		),
		cqrs.NewEventHandler(
			"ops_read_model.OnTicketBookingConfirmed",
			opsBookingsHandlers.OnTicketBookingConfirmed,
		),
		cqrs.NewEventHandler(
			"ops_read_model.OnTicketPrinted",
			opsBookingsHandlers.OnTicketPrinted,
		),
		cqrs.NewEventHandler(
			"ops_read_model.OnTicketRefunded",
			opsBookingsHandlers.OnTicketRefunded,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.OnVipBundleInitialized",
			vipBundleProcessManager.OnVipBundleInitialized,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.OnBookingMade",
			vipBundleProcessManager.OnBookingMade,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.OnTicketBookingConfirmed",
			vipBundleProcessManager.OnTicketBookingConfirmed,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.OnBookingFailed",
			vipBundleProcessManager.OnBookingFailed,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.OnFlightBooked",
			vipBundleProcessManager.OnFlightBooked,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.OnFlightBookingFailed",
			vipBundleProcessManager.OnFlightBookingFailed,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.OnTaxiBooked",
			vipBundleProcessManager.OnTaxiBooked,
		),
		cqrs.NewEventHandler(
			"vip_bundle_process_manager.OnTaxiBookingFailed",
			vipBundleProcessManager.OnTaxiBookingFailed,
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
		commandsHandler.BookShowTicketsHandler(),
		commandsHandler.BookFlightHandler(),
		commandsHandler.BookTaxiHandler(),
		commandsHandler.CancelFlightHandler(),
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

			return dataLake.StoreEvent(
				msg.Context(),
				entity.DataLakeEvent{
					ID:          event.Header.ID,
					PublishedAt: event.Header.PublishedAt,
					Name:        eventName,
					Payload:     msg.Payload,
				},
			)
		},
	)

	return router, nil
}
