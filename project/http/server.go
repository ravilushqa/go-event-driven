package http

import (
	"context"
	"errors"
	"net/http"

	echoHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"

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

type OpsBookingReadModel interface {
	AllReservations(receiptIssueDateFilter string) ([]entity.OpsBooking, error)
	ReservationReadModel(ctx context.Context, id string) (entity.OpsBooking, error)
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
	opsBookingReadModel   OpsBookingReadModel
}

func NewServer(
	addr string,
	eventbus *cqrs.EventBus,
	commandBus *cqrs.CommandBus,
	spreadsheetsAPIClient SpreadsheetsAPI,
	ticketsRepo TicketsRepository,
	showsRepo ShowsRepository,
	bookingsRepo BookingsRepository,
	opsBookingReadModel OpsBookingReadModel,
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
		opsBookingReadModel:   opsBookingReadModel,
	}

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	e.GET("/ops/bookings", server.GetOpsTickets)
	e.GET("/ops/bookings/:id", server.GetOpsTicket)
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
