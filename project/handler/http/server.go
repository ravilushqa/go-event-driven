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

type Server struct {
	eventbus              *cqrs.EventBus
	spreadsheetsAPIClient SpreadsheetsAPI
	repo                  TicketsRepository
	e                     *echo.Echo
	addr                  string
}

func NewServer(eventbus *cqrs.EventBus, spreadsheetsAPIClient SpreadsheetsAPI, repo TicketsRepository, addr string) *Server {
	e := echoHTTP.NewEcho()

	server := &Server{
		eventbus:              eventbus,
		spreadsheetsAPIClient: spreadsheetsAPIClient,
		repo:                  repo,
		addr:                  addr,
		e:                     e,
	}

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	e.POST("/tickets-status", server.PostTicketsStatus)

	e.GET("/tickets", server.GetTickets)

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
