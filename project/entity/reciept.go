package entity

import "time"

type IssueReceiptRequest struct {
	TicketID string
	Price    Money
}

type IssueReceiptResponse struct {
	ReceiptNumber string    `json:"number"`
	IssuedAt      time.Time `json:"issued_at"`
}
