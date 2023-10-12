package entity

import "time"

type IssueReceiptRequest struct {
	TicketID       string
	Price          Money
	IdempotencyKey string
}

type IssueReceiptResponse struct {
	ReceiptNumber string    `json:"number"`
	IssuedAt      time.Time `json:"issued_at"`
}
