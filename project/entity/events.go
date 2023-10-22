package entity

import (
	"time"

	"github.com/google/uuid"
)

type EventHeader struct {
	ID             string    `json:"id"`
	PublishedAt    time.Time `json:"published_at"`
	IdempotencyKey string    `json:"idempotency_key"`
}

func NewEventHeader() EventHeader {
	return EventHeader{
		ID:          uuid.NewString(),
		PublishedAt: time.Now().UTC(),
	}
}

func NewEventHeaderWithIdempotencyKey(idempotencyKey string) EventHeader {
	return EventHeader{
		ID:             uuid.NewString(),
		PublishedAt:    time.Now().UTC(),
		IdempotencyKey: idempotencyKey,
	}
}

type TicketBookingConfirmed_v1 struct {
	Header        EventHeader `json:"header"`
	TicketID      string      `json:"ticket_id"`
	CustomerEmail string      `json:"customer_email"`
	Price         Money       `json:"price"`
	BookingID     string      `json:"booking_id"`
}

type TicketBookingCanceled_v1 struct {
	Header        EventHeader `json:"header"`
	TicketID      string      `json:"ticket_id"`
	CustomerEmail string      `json:"customer_email"`
	Price         Money       `json:"price"`
	BookingID     string      `json:"booking_id"`
}

type TicketPrinted_v1 struct {
	Header   EventHeader `json:"header"`
	TicketID string      `json:"ticket_id"`
	FileName string      `json:"file_name"`
}

type BookingMade_v1 struct {
	Header          EventHeader `json:"header"`
	BookingID       string      `json:"booking_id"`
	NumberOfTickets int         `json:"number_of_tickets"`
	CustomerEmail   string      `json:"customer_email"`
	ShowID          string      `json:"show_id"`
}

type TicketReceiptIssued_v1 struct {
	Header        EventHeader `json:"header"`
	TicketID      string      `json:"ticket_id"`
	ReceiptNumber string      `json:"receipt_number"`
	IssuedAt      time.Time   `json:"issued_at"`
}

type TicketRefunded_v1 struct {
	Header   EventHeader `json:"header"`
	TicketID string      `json:"ticket_id"`
}
