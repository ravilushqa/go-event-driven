package pubsub

import (
	"context"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

func (h Handler) PrintTicketHandler() cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"PrintTicketHandler",
		func(ctx context.Context, event *entity.TicketBookingConfirmed) error {
			log.FromContext(ctx).Info("Printing ticket")
			return h.filesService.Put(ctx, *event)
		},
	)
}
