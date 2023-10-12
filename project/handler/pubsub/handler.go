package pubsub

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

//go:generate mockery --name SpreadsheetsAPI --output ../mocks --outpkg mocks --case underscore
type SpreadsheetsAPI interface {
	AppendRow(ctx context.Context, sheetName string, row []string) error
}

//go:generate mockery --name ReceiptsService --output ../mocks --outpkg mocks --case underscore
type ReceiptsService interface {
	IssueReceipt(ctx context.Context, request entity.IssueReceiptRequest) (entity.IssueReceiptResponse, error)
}

//go:generate mockery --name TicketsRepository --output ../../mocks --outpkg mocks --case underscore
type TicketsRepository interface {
	Store(ctx context.Context, ticket entity.Ticket) error
	Delete(ctx context.Context, ticketID string) error
}

//go:generate mockery --name FileService --output ../../mocks --outpkg mocks --case underscore
type FileService interface {
	Put(ctx context.Context, ticket entity.TicketBookingConfirmed) (string, error)
}

type Handler struct {
	spreadsheetsService SpreadsheetsAPI
	receiptsService     ReceiptsService
	ticketsRepository   TicketsRepository
	filesService        FileService
	eventbus            *cqrs.EventBus
}

func NewHandler(
	spreadsheetsService SpreadsheetsAPI,
	receiptsService ReceiptsService,
	ticketsRepository TicketsRepository,
	filesService FileService,
	eventbus *cqrs.EventBus,
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
	if filesService == nil {
		panic("missing filesService")
	}
	if eventbus == nil {
		panic("missing eventbus")
	}

	return Handler{
		spreadsheetsService: spreadsheetsService,
		receiptsService:     receiptsService,
		ticketsRepository:   ticketsRepository,
		filesService:        filesService,
		eventbus:            eventbus,
	}
}
