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

func (c DeadNationClient) PostTicketBooking(ctx context.Context, bookingID, customerAddress, eventID string, numberOfTickets int) error {
	bookingUUID, err := uuid.Parse(bookingID)
	if err != nil {
		return fmt.Errorf("failed to parse booking ID: %w", err)
	}

	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return fmt.Errorf("failed to parse event ID: %w", err)
	}
	resp, err := c.clients.DeadNation.PostTicketBookingWithResponse(ctx, dead_nation.PostTicketBookingRequest{
		BookingId:       bookingUUID,
		CustomerAddress: customerAddress,
		EventId:         eventUUID,
		NumberOfTickets: numberOfTickets,
	},
	)
	if err != nil {
		return fmt.Errorf("failed to post ticket booking: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code while posting ticket booking: %d", resp.StatusCode())
	}

	return nil
}
