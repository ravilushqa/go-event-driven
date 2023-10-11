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
			name, err := h.filesService.Put(ctx, *event)
			if err != nil {
				return err
			}

			ticketPrinter := entity.TicketPrinted{
				Header:   entity.NewEventHeader(),
				TicketID: event.TicketID,
				FileName: name,
			}

			return h.eventbus.Publish(ctx, ticketPrinter)
		},
	)
}
