package command

import (
	"context"

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
	receiptsService ReceiptsService
	paymentService  PaymentService
}

func NewHandler(
	receiptsService ReceiptsService,
	paymentService PaymentService,
) Handler {
	if receiptsService == nil {
		panic("missing receiptsService")
	}
	if paymentService == nil {
		panic("missing paymentService")
	}

	return Handler{
		receiptsService: receiptsService,
		paymentService:  paymentService,
	}
}
