package entity

type RefundTicket struct {
	Header   EventHeader
	TicketID string
}

type BookShowTickets struct {
	BookingID string `json:"booking_id"`

	CustomerEmail   string `json:"customer_email"`
	NumberOfTickets int    `json:"number_of_tickets"`
	ShowId          string `json:"show_id"`
}

type BookFlight struct {
	CustomerEmail  string   `json:"customer_email"`
	FlightID       string   `json:"to_flight_id"`
	Passengers     []string `json:"passengers"`
	ReferenceID    string   `json:"reference_id"`
	IdempotencyKey string   `json:"idempotency_key"`
}

type BookTaxi struct {
	CustomerEmail      string `json:"customer_email"`
	CustomerName       string `json:"customer_name"`
	NumberOfPassengers int    `json:"number_of_passengers"`
	ReferenceID        string `json:"reference_id"`
	IdempotencyKey     string `json:"idempotency_key"`
}

type CancelFlightTickets struct {
	FlightTicketIDs []string `json:"flight_ticket_id"`
}
