package entity

type Booking struct {
	BookingID       string `json:"booking_id" db:"booking_id"`
	ShowID          string `json:"show_id" db:"show_id"`
	NumberOfTickets int    `json:"number_of_tickets" db:"number_of_tickets"`
	CustomerEmail   string `json:"customer_email" db:"customer_email"`
}
