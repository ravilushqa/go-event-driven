package gateway

import (
	"context"
	"sync"

	"tickets/entity"
)

type PaymentMock struct {
	mock    sync.Mutex
	Refunds map[string]entity.RefundTicket
}

func (c *PaymentMock) PutRefundsWithResponse(ctx context.Context, command entity.RefundTicket) error {
	c.mock.Lock()
	defer c.mock.Unlock()
	if c.Refunds == nil {
		c.Refunds = make(map[string]entity.RefundTicket)
	}

	c.Refunds[command.TicketID] = command

	return nil
}
