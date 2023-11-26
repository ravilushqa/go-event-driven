package event

import (
	"context"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

func (h Handler) AppendToTrackerHandler() cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"AppendToTrackerHandler",
		func(ctx context.Context, event *entity.TicketBookingConfirmed_v1) error {
			log.FromContext(ctx).Info("Appending ticket to the tracker")
			return h.spreadsheetsService.AppendRow(
				ctx,
				"tickets-to-print",
				[]string{event.TicketID, event.CustomerEmail, event.Price.Amount, event.Price.Currency},
			)
		},
	)
}
