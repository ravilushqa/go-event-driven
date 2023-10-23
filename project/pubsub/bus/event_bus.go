package bus

import (
	"fmt"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"

	"tickets/entity"
)

func NewEventBus(pub message.Publisher) (*cqrs.EventBus, error) {
	return cqrs.NewEventBusWithConfig(pub, cqrs.EventBusConfig{
		GeneratePublishTopic: func(params cqrs.GenerateEventPublishTopicParams) (string, error) {
			event, ok := params.Event.(entity.Event)
			if !ok {
				return "", fmt.Errorf("invalid event type: %T doesn't implement entities.Event", params.Event)
			}

			if event.IsInternal() {
				// Publish directly to the per-event topic
				return "internal-events.svc-tickets." + params.EventName, nil
			} else {
				// Publish to the "events" topic, so it will be stored to the data lake and forwarded to the
				// per-event topic
				return "events", nil
			}
		},
		Marshaler: cqrs.JSONMarshaler{
			GenerateName: cqrs.StructName,
		},
	})
}
