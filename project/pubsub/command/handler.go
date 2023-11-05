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

type TransportationService interface {
	PutFlightTicketsWithResponse(ctx context.Context, bookFlight entity.BookFlight) ([]string, error)
	PutTaxiBookingWithResponse(ctx context.Context, bookTaxi entity.BookTaxi) (string, error)
}

type ShowsRepository interface {
	Store(ctx context.Context, show entity.Show) error
	Get(ctx context.Context, showID string) (entity.Show, error)
}

type BookingsRepository interface {
	Store(ctx context.Context, booking entity.Booking, showTicketsCount int) error
}

type Handler struct {
	eventBus        *cqrs.EventBus
	receiptsService ReceiptsService
	paymentService  PaymentService
	transService    TransportationService
	showsRepo       ShowsRepository
	bookingsRepo    BookingsRepository
}

func NewHandler(
	eventBus *cqrs.EventBus,
	receiptsService ReceiptsService,
	paymentService PaymentService,
	transService TransportationService,
	showRepo ShowsRepository,
	bookingRepo BookingsRepository,
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
	if showRepo == nil {
		panic("missing showRepo")
	}
	if bookingRepo == nil {
		panic("missing bookingRepo")
	}
	if transService == nil {
		panic("missing transService")
	}

	return Handler{
		eventBus:        eventBus,
		receiptsService: receiptsService,
		paymentService:  paymentService,
		transService:    transService,
		showsRepo:       showRepo,
		bookingsRepo:    bookingRepo,
	}
}
