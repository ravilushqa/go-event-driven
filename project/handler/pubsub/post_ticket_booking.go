package pubsub

import (
	"context"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

func (h Handler) PostTicketBookingHandler() cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"PostTicketBookingHandler",
		func(ctx context.Context, event *entity.BookingMade) error {
			log.FromContext(ctx).Info("Posting ticket booking to Dead Nation")

			show, err := h.showsRepo.Get(ctx, event.ShowID)
			if err != nil {
				return err
			}

			return h.deadNationService.PostTicketBooking(ctx, event.BookingID, event.CustomerEmail, show.DeadNationID, event.NumberOfTickets)
		},
	)
}
