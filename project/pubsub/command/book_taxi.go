package command

import (
	"context"
	"errors"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

func (h Handler) BookTaxiHandler() cqrs.CommandHandler {
	return cqrs.NewCommandHandler(
		"BookTaxiHandler",
		func(ctx context.Context, event *entity.BookTaxi) error {
			log.FromContext(ctx).Infof("BookTaxiHandler: %s", event.ReferenceID)
			bookingID, err := h.transService.PutTaxiBookingWithResponse(ctx, *event)
			if err != nil {
				if errors.Is(err, entity.ErrConflict) {
					return h.eventBus.Publish(ctx, entity.TaxiBookingFailed_v1{
						Header:        entity.NewEventHeader(),
						FailureReason: "conflict while booking taxi",
						ReferenceID:   event.ReferenceID,
					})
				}
				return err
			}

			return h.eventBus.Publish(ctx, entity.TaxiBooked_v1{
				Header:        entity.NewEventHeaderWithIdempotencyKey(event.IdempotencyKey),
				ReferenceID:   event.ReferenceID,
				TaxiBookingID: bookingID,
			})
		},
	)
}
