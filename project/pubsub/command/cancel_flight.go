package command

import (
	"context"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

func (h Handler) CancelFlightHandler() cqrs.CommandHandler {
	return cqrs.NewCommandHandler(
		"CancelFlightHandler",
		func(ctx context.Context, event *entity.CancelFlightTickets) error {
			log.FromContext(ctx).Infof("CancelFlightHandler: %s", event.FlightTicketIDs)
			return h.transService.DeleteFlightTicketsWithResponse(ctx, *event)
		},
	)
}
