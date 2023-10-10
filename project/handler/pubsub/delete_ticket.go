package pubsub

import (
	"context"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

func (h Handler) DeleteTicketHandler() cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"DeleteTicketHandler",
		func(ctx context.Context, event *entity.TicketBookingCanceled) error {
			log.FromContext(ctx).Info("Deleting ticket from DB")
			return h.ticketsRepository.Delete(ctx, event.TicketID)
		},
	)
}
