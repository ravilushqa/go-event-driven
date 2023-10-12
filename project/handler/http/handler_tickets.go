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

type ticketResponse struct {
	TicketID      string       `json:"ticket_id"`
	CustomerEmail string       `json:"customer_email"`
	Price         entity.Money `json:"price"`
}

func (s Server) PostTicketsStatus(c echo.Context) error {
	var request ticketsStatusRequest
	err := c.Bind(&request)
	if err != nil {
		return err
	}

	for _, ticket := range request.Tickets {
		header := entity.NewEventHeader()
		if idkey := c.Request().Header.Get("Idempotency-Key"); idkey != "" {
			header = entity.NewEventHeaderWithIdempotencyKey(fmt.Sprintf("%s-%s", ticket.TicketID, idkey))
		}
		if ticket.Status == "confirmed" {
			event := entity.TicketBookingConfirmed{
				Header:        header,
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
				Header:        header,
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

func (s Server) GetTickets(c echo.Context) error {
	tickets, err := s.repo.FindAll(c.Request().Context())
	if err != nil {
		return err
	}

	response := make([]ticketResponse, 0, len(tickets))
	for _, ticket := range tickets {
		response = append(response, ticketResponse{
			TicketID:      ticket.TicketID,
			CustomerEmail: ticket.CustomerEmail,
			Price: entity.Money{
				Amount:   ticket.PriceAmount,
				Currency: ticket.PriceCurrency,
			},
		})
	}

	return c.JSON(http.StatusOK, response)
}
