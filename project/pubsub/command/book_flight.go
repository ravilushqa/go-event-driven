package command

import (
	"context"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

func (h Handler) BookFlightHandler() cqrs.CommandHandler {
	return cqrs.NewCommandHandler(
		"BookFlightHandler",
		func(ctx context.Context, event *entity.BookFlight) error {
			log.FromContext(ctx).Infof("BookFlightHandler: %s", event.FlightID)
			ticketIDs, err := h.transService.PutFlightTicketsWithResponse(ctx, *event)
			if err != nil {
				return err
			}

			return h.eventBus.Publish(ctx, entity.FlightBooked_v1{
				Header:      entity.NewEventHeaderWithIdempotencyKey(event.IdempotencyKey),
				FlightID:    event.FlightID,
				ReferenceID: event.ReferenceID,
				TicketIDs:   ticketIDs,
			})
		},
	)
}
