package event

import (
	"context"
	"fmt"

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

			resp, err := h.receiptsService.IssueReceipt(ctx, request)
			if err != nil {
				return fmt.Errorf("failed to issue receipt: %w", err)
			}

			return h.eventbus.Publish(ctx, entity.TicketReceiptIssued{
				Header:        entity.NewEventHeaderWithIdempotencyKey(event.Header.IdempotencyKey),
				TicketID:      event.TicketID,
				ReceiptNumber: resp.ReceiptNumber,
				IssuedAt:      resp.IssuedAt,
			})
		},
	)
}
