package pubsub

import (
	"context"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

func (h Handler) IssueReceiptHandler() cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"IssueReceiptHandler",
		func(ctx context.Context, event *entity.TicketBookingConfirmed) error {
			log.FromContext(ctx).Info("Issuing receipt")
			request := entity.IssueReceiptRequest{
				TicketID:       event.TicketID,
				Price:          event.Price,
				IdempotencyKey: event.Header.IdempotencyKey,
			}

			_, err := h.receiptsService.IssueReceipt(ctx, request)
			return err
		},
	)
}
