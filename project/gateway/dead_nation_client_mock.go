package gateway

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

type DeadNationMock struct {
	lock     sync.Mutex
	bookings map[uuid.UUID]BookingData
}

type BookingData struct {
	BookingID       uuid.UUID
	CustomerAddress string
	EventID         uuid.UUID
	NumberOfTickets int
}

func (c *DeadNationMock) PostTicketBooking(_ context.Context, bookingID, customerAddress, eventID string, numberOfTickets int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.bookings == nil {
		c.bookings = make(map[uuid.UUID]BookingData)
	}

	bookingUUID, err := uuid.Parse(bookingID)
	if err != nil {
		return fmt.Errorf("could not parse booking uuid: %w", err)
	}
	eventUUID, err := uuid.Parse(eventID)
	if err != nil {
		return fmt.Errorf("could not parse event uuid: %w", err)
	}

	bData := BookingData{
		BookingID:       bookingUUID,
		CustomerAddress: customerAddress,
		EventID:         eventUUID,
		NumberOfTickets: numberOfTickets,
	}

	c.bookings[bookingUUID] = bData

	return nil
}
