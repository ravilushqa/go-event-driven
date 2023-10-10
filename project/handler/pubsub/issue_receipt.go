package pubsub

import (
	"context"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

type ReceiptsService interface {
	IssueReceipt(ctx context.Context, request entity.IssueReceiptRequest) (entity.IssueReceiptResponse, error)
}

func IssueReceiptHandler(rs ReceiptsService) cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"IssueReceiptHandler",
		func(ctx context.Context, event *entity.TicketBookingConfirmed) error {
			log.FromContext(ctx).Info("Issuing receipt")
			request := entity.IssueReceiptRequest{
				TicketID: event.TicketID,
				Price:    event.Price,
			}

			_, err := rs.IssueReceipt(ctx, request)
			return err
		},
	)
}
