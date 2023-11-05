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
