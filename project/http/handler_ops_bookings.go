package http

import (
	"github.com/labstack/echo/v4"
)

func (s Server) GetOpsBookings(c echo.Context) error {
	bookings, err := s.opsBookingsRepo.FindAll(c.Request().Context())
	if err != nil {
		return err
	}

	return c.JSON(200, bookings)
}

func (s Server) GetOpsBooking(c echo.Context) error {
	ticketID := c.Param("id")
	if ticketID == "" {
		return echo.NewHTTPError(400, "missing ticket_id")
	}
	bookings, err := s.opsBookingsRepo.Get(c.Request().Context(), ticketID)
	if err != nil {
		return err
	}

	return c.JSON(200, bookings)
}
