package pubsub

import (
	"context"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

type TicketsRepository interface {
	Store(ctx context.Context, ticket entity.Ticket) error
}

func StoreDBHandler(r TicketsRepository) cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"StoreDBHandler",
		func(ctx context.Context, event *entity.TicketBookingConfirmed) error {
			log.FromContext(ctx).Info("Storing ticket in DB")
			return r.Store(ctx, entity.Ticket{
				ID:            event.TicketID,
				PriceAmount:   event.Price.Amount,
				PriceCurrency: event.Price.Currency,
				CustomerEmail: event.CustomerEmail,
			})
		},
	)
}
