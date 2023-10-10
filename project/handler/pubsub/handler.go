package pubsub

import (
	"context"

	"tickets/entity"
)

type SpreadsheetsAPI interface {
	AppendRow(ctx context.Context, sheetName string, row []string) error
}

type ReceiptsService interface {
	IssueReceipt(ctx context.Context, request entity.IssueReceiptRequest) (entity.IssueReceiptResponse, error)
}

type TicketsRepository interface {
	Store(ctx context.Context, ticket entity.Ticket) error
	Delete(ctx context.Context, ticketID string) error
}

type Handler struct {
	spreadsheetsService SpreadsheetsAPI
	receiptsService     ReceiptsService
	ticketsRepository   TicketsRepository
}

func NewHandler(
	spreadsheetsService SpreadsheetsAPI,
	receiptsService ReceiptsService,
	ticketsRepository TicketsRepository,
) Handler {
	if spreadsheetsService == nil {
		panic("missing spreadsheetsService")
	}
	if receiptsService == nil {
		panic("missing receiptsService")
	}
	if ticketsRepository == nil {
		panic("missing ticketsRepository")
	}

	return Handler{
		spreadsheetsService: spreadsheetsService,
		receiptsService:     receiptsService,
		ticketsRepository:   ticketsRepository,
	}
}
