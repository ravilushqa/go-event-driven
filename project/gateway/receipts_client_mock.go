package gateway

import (
	"context"
	"sync"
	"time"

	"tickets/entity"
)

type ReceiptsMock struct {
	mock sync.Mutex

	IssuedReceipts map[string]entity.IssueReceiptRequest
	VoidedReceipts map[string]entity.RefundTicket
}

func (c *ReceiptsMock) IssueReceipt(ctx context.Context, request entity.IssueReceiptRequest) (entity.IssueReceiptResponse, error) {
	c.mock.Lock()
	defer c.mock.Unlock()

	if c.IssuedReceipts == nil {
		c.IssuedReceipts = make(map[string]entity.IssueReceiptRequest)
	}

	c.IssuedReceipts[request.IdempotencyKey] = request

	return entity.IssueReceiptResponse{
		ReceiptNumber: "mocked-receipt-number",
		IssuedAt:      time.Now(),
	}, nil
}

func (c *ReceiptsMock) PutVoidReceiptWithResponse(ctx context.Context, command entity.RefundTicket) error {
	c.mock.Lock()
	defer c.mock.Unlock()

	if c.VoidedReceipts == nil {
		c.VoidedReceipts = make(map[string]entity.RefundTicket)
	}

	c.VoidedReceipts[command.Header.IdempotencyKey] = command

	return nil
}
