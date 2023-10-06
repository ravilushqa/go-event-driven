package http

import (
	"net/http"

	libHttp "github.com/ThreeDotsLabs/go-event-driven/common/http"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/labstack/echo/v4"
)

func NewHttpRouter(publisher message.Publisher, spreadsheetsAPIClient SpreadsheetsAPI) *echo.Echo {
	e := libHttp.NewEcho()

	handler := Handler{
		publisher:             publisher,
		spreadsheetsAPIClient: spreadsheetsAPIClient,
	}

	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	e.POST("/tickets-status", handler.PostTicketsStatus)

	return e
}
