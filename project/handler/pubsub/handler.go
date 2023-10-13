package pubsub

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"

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

type ShowsRepository interface {
	Store(ctx context.Context, show entity.Show) error
	Get(ctx context.Context, showID string) (entity.Show, error)
}

type FileService interface {
	UploadFile(ctx context.Context, fileID string, fileContent string) error
	DownloadFile(ctx context.Context, fileID string) (string, error)
}

type DeadNationService interface {
	PostTicketBooking(ctx context.Context, BookingID, CustomerAddress, EventID string, NumberOfTickets int) error
}

type Handler struct {
	eventbus            *cqrs.EventBus
	spreadsheetsService SpreadsheetsAPI
	receiptsService     ReceiptsService
	filesService        FileService
	deadNationService   DeadNationService
	ticketsRepository   TicketsRepository
	showsRepo           ShowsRepository
}

func NewHandler(
	eventbus *cqrs.EventBus,
	spreadsheetsService SpreadsheetsAPI,
	receiptsService ReceiptsService,
	filesService FileService,
	deadNationService DeadNationService,
	ticketsRepository TicketsRepository,
	showsRepo ShowsRepository,
) Handler {
	if eventbus == nil {
		panic("missing eventbus")
	}
	if spreadsheetsService == nil {
		panic("missing spreadsheetsService")
	}
	if receiptsService == nil {
		panic("missing receiptsService")
	}
	if filesService == nil {
		panic("missing filesService")
	}
	if deadNationService == nil {
		panic("missing deadNationService")
	}
	if ticketsRepository == nil {
		panic("missing ticketsRepository")
	}
	if showsRepo == nil {
		panic("missing showsRepo")
	}

	return Handler{
		eventbus:            eventbus,
		spreadsheetsService: spreadsheetsService,
		receiptsService:     receiptsService,
		filesService:        filesService,
		deadNationService:   deadNationService,
		ticketsRepository:   ticketsRepository,
		showsRepo:           showsRepo,
	}
}
