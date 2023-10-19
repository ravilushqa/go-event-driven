package http

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"

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

type postBookTicketsRequest struct {
	ShowID          string `json:"show_id"`
	NumberOfTickets int    `json:"number_of_tickets"`
	CustomerEmail   string `json:"customer_email"`
}

type postBookTicketsResponse struct {
	BookingID string `json:"booking_id"`
}

func (s Server) PostTicketsStatus(c echo.Context) error {
	var request ticketsStatusRequest
	err := c.Bind(&request)
	if err != nil {
		return err
	}

	idempotencyKey := c.Request().Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Idempotency-Key header is required")
	}

	for _, ticket := range request.Tickets {
		if ticket.Status == "confirmed" {
			event := entity.TicketBookingConfirmed{
				Header:        entity.NewEventHeaderWithIdempotencyKey(idempotencyKey + ticket.TicketID),
				TicketID:      ticket.TicketID,
				CustomerEmail: ticket.CustomerEmail,
				Price:         ticket.Price,
				BookingID:     ticket.BookingID,
			}

			err = s.eventbus.Publish(c.Request().Context(), event)
			if err != nil {
				return err
			}
		} else if ticket.Status == "canceled" {
			event := entity.TicketBookingCanceled{
				Header:        entity.NewEventHeaderWithIdempotencyKey(idempotencyKey + ticket.TicketID),
				TicketID:      ticket.TicketID,
				CustomerEmail: ticket.CustomerEmail,
				Price:         ticket.Price,
				BookingID:     ticket.BookingID,
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
	tickets, err := s.ticketsRepo.FindAll(c.Request().Context())
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

func (s Server) TicketRefund(c echo.Context) error {
	ticketID := c.Param("ticket_id")

	err := s.commandBus.Send(c.Request().Context(), &entity.RefundTicket{
		Header:   entity.NewEventHeaderWithIdempotencyKey(uuid.NewString()),
		TicketID: ticketID,
	})
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusAccepted)
}
