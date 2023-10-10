package pubsub

import (
	"context"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

func (h Handler) StoreTicketHandler() cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"StoreTicketHandler",
		func(ctx context.Context, event *entity.TicketBookingConfirmed) error {
			log.FromContext(ctx).Info("Storing ticket in DB")
			return h.ticketsRepository.Store(ctx, entity.Ticket{
				TicketID:      event.TicketID,
				PriceAmount:   event.Price.Amount,
				PriceCurrency: event.Price.Currency,
				CustomerEmail: event.CustomerEmail,
			})
		},
	)
}
