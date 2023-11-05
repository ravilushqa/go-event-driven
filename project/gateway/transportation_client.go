package gateway

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/transportation"
	"github.com/google/uuid"

	"tickets/entity"
)

type TransportationClient struct {
	clients *clients.Clients
}

func NewTransportationClient(clients *clients.Clients) TransportationClient {
	return TransportationClient{
		clients: clients,
	}
}

func (c TransportationClient) PutFlightTicketsWithResponse(ctx context.Context, bookFlight entity.BookFlight) ([]string, error) {
	FlightID, err := uuid.Parse(bookFlight.FlightID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse flight id: %w", err)
	}

	resp, err := c.clients.Transportation.PutFlightTicketsWithResponse(ctx, transportation.BookFlightTicketRequest{
		CustomerEmail:  bookFlight.CustomerEmail,
		FlightId:       FlightID,
		PassengerNames: bookFlight.Passengers,
		ReferenceId:    bookFlight.ReferenceID,
		IdempotencyKey: bookFlight.IdempotencyKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to flight tickets booking: %w", err)
	}

	if resp.StatusCode() == http.StatusConflict {
		return nil, entity.ErrConflict
	}

	if resp.StatusCode() != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code while flight tickets booking: %d", resp.StatusCode())
	}

	var ticketIDs []string
	for _, ticket := range resp.JSON201.TicketIds {
		ticketIDs = append(ticketIDs, ticket.String())
	}

	return ticketIDs, nil
}

func (c TransportationClient) PutTaxiBookingWithResponse(ctx context.Context, bookTaxi entity.BookTaxi) (string, error) {
	resp, err := c.clients.Transportation.PutTaxiBookingWithResponse(ctx, transportation.TaxiBookingRequest{
		CustomerEmail:      bookTaxi.CustomerEmail,
		NumberOfPassengers: bookTaxi.NumberOfPassengers,
		PassengerName:      bookTaxi.CustomerName,
		ReferenceId:        bookTaxi.ReferenceID,
		IdempotencyKey:     bookTaxi.IdempotencyKey,
	})
	if err != nil {
		return "", fmt.Errorf("failed to taxi booking: %w", err)
	}

	if resp.StatusCode() != http.StatusCreated {
		return "", fmt.Errorf("unexpected status code while taxi booking: %d", resp.StatusCode())
	}

	return resp.JSON201.BookingId.String(), nil
}

func (c TransportationClient) DeleteFlightTicketsWithResponse(ctx context.Context, cancelFlight entity.CancelFlightTickets) error {
	var ticketIDs []uuid.UUID
	for _, ticketID := range cancelFlight.FlightTicketIDs {
		id, err := uuid.Parse(ticketID)
		if err != nil {
			return fmt.Errorf("failed to parse ticket id: %w", err)
		}

		ticketIDs = append(ticketIDs, id)
	}

	for _, ticketID := range ticketIDs {
		resp, err := c.clients.Transportation.DeleteFlightTicketsTicketIdWithResponse(ctx, ticketID)
		if err != nil {
			return fmt.Errorf("failed to cancel flight tickets: %w", err)
		}

		if resp.StatusCode() != http.StatusNoContent {
			return fmt.Errorf("unexpected status code while canceling flight tickets: %d", resp.StatusCode())
		}
	}

	return nil
}
