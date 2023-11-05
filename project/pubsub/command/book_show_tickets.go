package command

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

func (h Handler) BookShowTicketsHandler() cqrs.CommandHandler {
	return cqrs.NewCommandHandler(
		"BookShowTicketsHandler",
		func(ctx context.Context, event *entity.BookShowTickets) error {
			log.FromContext(ctx).Infof("BookShowTicketsHandler: %s", event.BookingID)

			booking := entity.Booking{
				BookingID:       event.BookingID,
				ShowID:          event.ShowId,
				NumberOfTickets: event.NumberOfTickets,
				CustomerEmail:   event.CustomerEmail,
			}

			show, err := h.showsRepo.Get(ctx, event.ShowId)
			if err != nil {
				return fmt.Errorf("could not get show: %w", err)
			}

			err = h.bookingsRepo.Store(ctx, booking, show.NumberOfTickets)
			return err
		},
	)
}
