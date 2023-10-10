package pubsub

import (
	"context"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

type SpreadsheetsAPI interface {
	AppendRow(ctx context.Context, sheetName string, row []string) error
}

func AppendToTrackerHandler(sa SpreadsheetsAPI) cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"AppendToTrackerHandler",
		func(ctx context.Context, event *entity.TicketBookingConfirmed) error {
			log.FromContext(ctx).Info("Appending ticket to the tracker")
			return sa.AppendRow(
				ctx,
				"tickets-to-print",
				[]string{event.TicketID, event.CustomerEmail, event.Price.Amount, event.Price.Currency},
			)
		},
	)
}
