package gateway

import (
	"context"
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

	bookingUUID := uuid.NewSHA1(uuid.Nil, []byte(BookingID))
	eventUUID := uuid.NewSHA1(uuid.Nil, []byte(EventID))

	bData := BookingData{
		BookingID:       bookingUUID,
		CustomerAddress: CustomerAddress,
		EventID:         eventUUID,
		NumberOfTickets: NumberOfTickets,
	}

	c.bookings[bookingUUID] = bData

	return nil
}
