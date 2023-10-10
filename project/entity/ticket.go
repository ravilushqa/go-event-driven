package entity

type Ticket struct {
	ID            string `json:"ticket_id" db:"ticket_id"`
	PriceAmount   string `json:"price_amount" db:"price_amount"`
	PriceCurrency string `json:"price_currency" db:"price_currency"`
	CustomerEmail string `json:"customer_email" db:"customer_email"`
}
