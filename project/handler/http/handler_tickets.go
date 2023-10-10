package http

import (
	"fmt"
	"net/http"

	"tickets/entity"

	"github.com/labstack/echo/v4"
)

type ticketsStatusRequest struct {
	Tickets []ticketStatusRequest `json:"tickets"`
}

type ticketStatusRequest struct {
	TicketID      string       `json:"ticket_id"`
	Status        string       `json:"status"`
	Price         entity.Money `json:"price"`
	CustomerEmail string       `json:"customer_email"`
	BookingID     string       `json:"booking_id"`
}

func (s Server) PostTicketsStatus(c echo.Context) error {
	var request ticketsStatusRequest
	err := c.Bind(&request)
	if err != nil {
		return err
	}

	for _, ticket := range request.Tickets {
		if ticket.Status == "confirmed" {
			event := entity.TicketBookingConfirmed{
				Header:        entity.NewEventHeader(),
				TicketID:      ticket.TicketID,
				CustomerEmail: ticket.CustomerEmail,
				Price:         ticket.Price,
			}

			err = s.eventbus.Publish(c.Request().Context(), event)
			if err != nil {
				return err
			}
		} else if ticket.Status == "canceled" {
			event := entity.TicketBookingCanceled{
				Header:        entity.NewEventHeader(),
				TicketID:      ticket.TicketID,
				CustomerEmail: ticket.CustomerEmail,
				Price:         ticket.Price,
			}

			err = s.eventbus.Publish(c.Request().Context(), event)
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unknown ticket status: %s", ticket.Status)
		}
	}

	return c.NoContent(http.StatusOK)
}
