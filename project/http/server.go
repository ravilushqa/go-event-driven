package http

import (
	"context"
	"errors"
	"net/http"

	echoHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/labstack/echo/v4"

	"tickets/entity"
)

type SpreadsheetsAPI interface {
	AppendRow(ctx context.Context, spreadsheetName string, row []string) error
}

type TicketsRepository interface {
	FindAll(ctx context.Context) ([]entity.Ticket, error)
}

type ShowsRepository interface {
	Store(ctx context.Context, show entity.Show) error
	Get(ctx context.Context, showID string) (entity.Show, error)
}

type BookingsRepository interface {
	Store(ctx context.Context, booking entity.Booking, showTicketsCount int) error
}

type OpsBookingsRepository interface {
	FindAll(ctx context.Context) ([]entity.OpsBooking, error)
	Get(ctx context.Context, bookingID string) (entity.OpsBooking, error)
}

type Server struct {
	addr                  string
	e                     *echo.Echo
	eventbus              *cqrs.EventBus
	commandBus            *cqrs.CommandBus
	spreadsheetsAPIClient SpreadsheetsAPI
	ticketsRepo           TicketsRepository
	showsRepo             ShowsRepository
	bookingsRepo          BookingsRepository
	opsBookingsRepo       OpsBookingsRepository
}

func NewServer(
	addr string,
	eventbus *cqrs.EventBus,
	commandBus *cqrs.CommandBus,
	spreadsheetsAPIClient SpreadsheetsAPI,
	ticketsRepo TicketsRepository,
	showsRepo ShowsRepository,
	bookingsRepo BookingsRepository,
	opsBookingsRepo OpsBookingsRepository,
) *Server {
	e := echoHTTP.NewEcho()

	server := &Server{
		addr:                  addr,
		e:                     e,
		eventbus:              eventbus,
		commandBus:            commandBus,
		spreadsheetsAPIClient: spreadsheetsAPIClient,
		ticketsRepo:           ticketsRepo,
		showsRepo:             showsRepo,
		bookingsRepo:          bookingsRepo,
		opsBookingsRepo:       opsBookingsRepo,
	}

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	e.GET("/ops/bookings", server.GetOpsBookings)
	e.GET("/ops/bookings/:id", server.GetOpsBooking)
	e.GET("/tickets", server.GetTickets)
	e.POST("/tickets-status", server.PostTicketsStatus)
	e.PUT("/ticket-refund/:ticket_id", server.TicketRefund)
	e.POST("/book-tickets", server.PostBookTickets)

	e.POST("/shows", server.PostShows)

	return server
}

func (s Server) Run(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		err := s.e.Shutdown(ctx)
		if err != nil {
			log.FromContext(ctx).WithError(err).Error("failed to shutdown HTTP server")
		}
	}()
	log.FromContext(ctx).WithField("addr", s.addr).Info("[HTTP] server listening")
	if err := s.e.Start(s.addr); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
