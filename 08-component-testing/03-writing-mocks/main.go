package main

import (
	"context"
	"time"
)

type IssueReceiptRequest struct {
	TicketID string `json:"ticket_id"`
	Price    Money  `json:"price"`
}

type Money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type IssueReceiptResponse struct {
	ReceiptNumber string    `json:"number"`
	IssuedAt      time.Time `json:"issued_at"`
}

type ReceiptsServiceMock struct {
	IssuedReceipts []IssueReceiptRequest
}

func (m *ReceiptsServiceMock) IssueReceipt(_ context.Context, request IssueReceiptRequest) (IssueReceiptResponse, error) {
	m.IssuedReceipts = append(m.IssuedReceipts, request)
	return IssueReceiptResponse{
		ReceiptNumber: request.TicketID,
		IssuedAt:      time.Now(),
	}, nil
}
