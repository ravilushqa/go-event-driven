package gateway

import (
	"context"
	"sync"

	"tickets/entity"
)

type TransportationMock struct {
	mock sync.Mutex

	BookedFlightTickets    map[string]entity.BookFlight
	BookedTaxiBookings     map[string]entity.BookTaxi
	CancelledFlightTickets map[string]entity.CancelFlightTickets
}

func (c *TransportationMock) PutFlightTicketsWithResponse(ctx context.Context, bookFlight entity.BookFlight) ([]string, error) {
	c.mock.Lock()
	defer c.mock.Unlock()

	if c.BookedFlightTickets == nil {
		c.BookedFlightTickets = make(map[string]entity.BookFlight)
	}

	c.BookedFlightTickets[bookFlight.FlightID] = bookFlight

	return []string{"mocked-flight-ticket-id"}, nil
}

func (c *TransportationMock) PutTaxiBookingWithResponse(ctx context.Context, bookTaxi entity.BookTaxi) (string, error) {
	c.mock.Lock()
	defer c.mock.Unlock()

	if c.BookedTaxiBookings == nil {
		c.BookedTaxiBookings = make(map[string]entity.BookTaxi)
	}

	c.BookedTaxiBookings[bookTaxi.ReferenceID] = bookTaxi

	return "mocked-taxi-booking-id", nil
}

func (c *TransportationMock) DeleteFlightTicketsWithResponse(ctx context.Context, cancelFlight entity.CancelFlightTickets) error {
	c.mock.Lock()
	defer c.mock.Unlock()

	if c.CancelledFlightTickets == nil {
		c.CancelledFlightTickets = make(map[string]entity.CancelFlightTickets)
	}

	for _, ticketID := range cancelFlight.FlightTicketIDs {
		c.CancelledFlightTickets[ticketID] = cancelFlight
	}

	return nil
}
