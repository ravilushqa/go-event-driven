package event

import (
	"context"

	"tickets/entity"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
)

func (h Handler) AppendToTracker(ctx context.Context, event entity.TicketBookingConfirmed) error {
	log.FromContext(ctx).Info("Appending ticket to the tracker")

	return h.spreadsheetsService.AppendRow(
		ctx,
		"tickets-to-print",
		[]string{event.TicketID, event.CustomerEmail, event.Price.Amount, event.Price.Currency},
	)
}
