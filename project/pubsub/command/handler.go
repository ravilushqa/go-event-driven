package command

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

type ReceiptsService interface {
	IssueReceipt(ctx context.Context, request entity.IssueReceiptRequest) (entity.IssueReceiptResponse, error)
	PutVoidReceiptWithResponse(ctx context.Context, command entity.RefundTicket) error
}

type PaymentService interface {
	PutRefundsWithResponse(ctx context.Context, command entity.RefundTicket) error
}

type Handler struct {
	eventBus        *cqrs.EventBus
	receiptsService ReceiptsService
	paymentService  PaymentService
}

func NewHandler(
	eventBus *cqrs.EventBus,
	receiptsService ReceiptsService,
	paymentService PaymentService,
) Handler {
	if eventBus == nil {
		panic("missing eventBus")
	}
	if receiptsService == nil {
		panic("missing receiptsService")
	}
	if paymentService == nil {
		panic("missing paymentService")
	}

	return Handler{
		eventBus:        eventBus,
		receiptsService: receiptsService,
		paymentService:  paymentService,
	}
}
