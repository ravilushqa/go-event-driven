package pubsub

import (
	"context"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

func (h Handler) RefundTicketHandler() cqrs.CommandHandler {
	return cqrs.NewCommandHandler(
		"RefundTicketHandler",
		func(ctx context.Context, event *entity.RefundTicket) error {
			log.FromContext(ctx).Infof("RefundTicketHandler: %s", event.TicketID)

			if err := h.receiptsService.PutVoidReceiptWithResponse(ctx, *event); err != nil {
				return err
			}

			if err := h.paymentService.PutRefundsWithResponse(ctx, *event); err != nil {
				return err
			}

			return nil
		},
	)
}
