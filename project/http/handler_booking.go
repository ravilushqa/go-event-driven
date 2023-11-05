package http

import (
	"errors"
	"fmt"
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

	show, err := s.showsRepo.Get(c.Request().Context(), request.ShowID)
	if err != nil {
		return fmt.Errorf("could not get show: %w", err)
	}

	err = s.bookingsRepo.Store(c.Request().Context(), booking, show.NumberOfTickets)
	if err != nil {
		if errors.Is(err, entity.ErrNoAvailableTickets) {
			return echo.NewHTTPError(http.StatusBadRequest, "not enough seats available")
		}

		return fmt.Errorf("could not store booking: %w", err)
	}

	return c.JSON(http.StatusCreated, postBookTicketsResponse{
		BookingID: booking.BookingID,
	})
}
