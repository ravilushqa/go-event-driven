package pubsub

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"

	"tickets/pubsub/command"
	"tickets/pubsub/event"
	"tickets/pubsub/outbox"
)

func NewWatermillRouter(
	postgresSubscriber message.Subscriber,
	publisher message.Publisher,
	eventProcessorConfig cqrs.EventProcessorConfig,
	eventHandler event.Handler,
	commandProcessorConfig cqrs.CommandProcessorConfig,
	commandsHandler command.Handler,
	watermillLogger watermill.LoggerAdapter,
) (*message.Router, error) {
	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return nil, fmt.Errorf("could not create router: %w", err)
	}

	useMiddlewares(router, watermillLogger)

	outbox.AddForwarderHandler(postgresSubscriber, publisher, router, watermillLogger)

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

	return router, nil
}
