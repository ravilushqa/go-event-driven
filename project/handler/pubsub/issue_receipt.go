package pubsub

import (
	"context"
	"fmt"

	"tickets/entity"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
)

func (h Handler) IssueReceipt(ctx context.Context, event entity.TicketBookingConfirmed) error {
	log.FromContext(ctx).Info("Issuing receipt")

	request := entity.IssueReceiptRequest{
		TicketID: event.TicketID,
		Price:    event.Price,
	}

	_, err := h.receiptsService.IssueReceipt(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to issue receipt: %w", err)
	}

	return nil
}
