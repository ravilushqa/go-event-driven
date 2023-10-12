package http

import (
	"context"
	"errors"
	"net/http"

	echoHTTP "github.com/ThreeDotsLabs/go-event-driven/common/http"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/jmoiron/sqlx"
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
}

type BookingsRepository interface {
	Store(ctx context.Context, booking entity.Booking) error
}

type Server struct {
	addr                  string
	e                     *echo.Echo
	db                    *sqlx.DB
	eventbus              *cqrs.EventBus
	txEventbus            *cqrs.EventBus
	spreadsheetsAPIClient SpreadsheetsAPI
	ticketsRepo           TicketsRepository
	showsRepo             ShowsRepository
	bookingsRepo          BookingsRepository
}

func NewServer(
	addr string,
	db *sqlx.DB,
	eventbus *cqrs.EventBus,
	txEventbus *cqrs.EventBus,
	spreadsheetsAPIClient SpreadsheetsAPI,
	ticketsRepo TicketsRepository,
	showsRepo ShowsRepository,
	bookingsRepo BookingsRepository,
) *Server {
	e := echoHTTP.NewEcho()

	server := &Server{
		addr:                  addr,
		db:                    db,
		e:                     e,
		eventbus:              eventbus,
		txEventbus:            txEventbus,
		spreadsheetsAPIClient: spreadsheetsAPIClient,
		ticketsRepo:           ticketsRepo,
		showsRepo:             showsRepo,
		bookingsRepo:          bookingsRepo,
	}

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	e.GET("/tickets", server.GetTickets)
	e.POST("/tickets-status", server.PostTicketsStatus)
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
