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

func (c *DeadNationMock) PostTicketBooking(_ context.Context, BookingID, CustomerAddress, EventID string, NumberOfTickets int) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.bookings == nil {
		c.bookings = make(map[uuid.UUID]BookingData)
	}

	bookingUUID, err := uuid.Parse(BookingID)
	if err != nil {
		return fmt.Errorf("could not parse booking uuid: %w", err)
	}
	eventUUID, err := uuid.Parse(EventID)
	if err != nil {
		return fmt.Errorf("could not parse event uuid: %w", err)
	}

	bData := BookingData{
		BookingID:       bookingUUID,
		CustomerAddress: CustomerAddress,
		EventID:         eventUUID,
		NumberOfTickets: NumberOfTickets,
	}

	c.bookings[bookingUUID] = bData

	return nil
}
