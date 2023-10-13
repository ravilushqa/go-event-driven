package gateway

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/dead_nation"
	"github.com/google/uuid"
)

type DeadNationClient struct {
	clients *clients.Clients
}

func NewDeadNationClient(clients *clients.Clients) DeadNationClient {
	return DeadNationClient{
		clients: clients,
	}
}

func (c DeadNationClient) PostTicketBooking(ctx context.Context, BookingID, CustomerAddress, EventID string, NumberOfTickets int) error {
	bookingID, err := uuid.Parse(BookingID)
	if err != nil {
		return fmt.Errorf("failed to parse booking ID: %w", err)
	}

	eventID, err := uuid.Parse(EventID)
	if err != nil {
		return fmt.Errorf("failed to parse event ID: %w", err)
	}
	resp, err := c.clients.DeadNation.PostTicketBookingWithResponse(ctx, dead_nation.PostTicketBookingJSONRequestBody{
		BookingId:       bookingID,
		CustomerAddress: CustomerAddress,
		EventId:         eventID,
		NumberOfTickets: NumberOfTickets,
	})
	if err != nil {
		return fmt.Errorf("failed to post ticket booking: %w", err)
	}

	if resp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("unexpected status code while posting ticket booking: %d", resp.StatusCode())
	}

	return nil
}
