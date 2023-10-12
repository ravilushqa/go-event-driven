package http

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"tickets/entity"
)

func (s Server) PostBookTickets(c echo.Context) error {
	var request postBookTicketsRequest
	err := c.Bind(&request)
	if err != nil {
		return err
	}

	booking := entity.Booking{
		BookingID:       uuid.NewString(),
		ShowID:          request.ShowID,
		NumberOfTickets: request.NumberOfTickets,
		CustomerEmail:   request.CustomerEmail,
	}

	err = s.bookingsRepo.Store(c.Request().Context(), booking)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, postBookTicketsResponse{
		BookingID: booking.BookingID,
	})
}
