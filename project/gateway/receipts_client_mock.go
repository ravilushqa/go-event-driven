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
}

func (c *ReceiptsMock) IssueReceipt(ctx context.Context, request entity.IssueReceiptRequest) (entity.IssueReceiptResponse, error) {
	c.mock.Lock()
	defer c.mock.Unlock()

	c.IssuedReceipts[request.IdempotencyKey] = request

	return entity.IssueReceiptResponse{
		ReceiptNumber: "mocked-receipt-number",
		IssuedAt:      time.Now(),
	}, nil
}
