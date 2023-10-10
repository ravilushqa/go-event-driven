package pubsub

import (
	"context"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

func CancelTicketHandler(sa SpreadsheetsAPI) cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"CancelTicketHandler",
		func(ctx context.Context, event *entity.TicketBookingConfirmed) error {
			log.FromContext(ctx).Info("Adding ticket refund to sheet")
			return sa.AppendRow(
				ctx,
				"tickets-to-refund",
				[]string{event.TicketID, event.CustomerEmail, event.Price.Amount, event.Price.Currency},
			)
		},
	)
}
