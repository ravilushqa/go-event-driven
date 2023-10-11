package entity

import (
	"time"

	"github.com/google/uuid"
)

type EventHeader struct {
	ID          string `json:"id"`
	PublishedAt string `json:"published_at"`
}

func NewEventHeader() EventHeader {
	return EventHeader{
		ID:          uuid.NewString(),
		PublishedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

type TicketBookingConfirmed struct {
	Header        EventHeader `json:"header"`
	TicketID      string      `json:"ticket_id"`
	CustomerEmail string      `json:"customer_email"`
	Price         Money       `json:"price"`
}

type TicketBookingCanceled struct {
	Header        EventHeader `json:"header"`
	TicketID      string      `json:"ticket_id"`
	CustomerEmail string      `json:"customer_email"`
	Price         Money       `json:"price"`
}

type TicketPrinted struct {
	Header   EventHeader `json:"header"`
	TicketID string      `json:"ticket_id"`
	FileName string      `json:"file_name"`
}
