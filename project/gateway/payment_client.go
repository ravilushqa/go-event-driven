package gateway

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/payments"

	"tickets/entity"
)

type PaymentClient struct {
	clients *clients.Clients
}

func NewPaymentClient(clients *clients.Clients) PaymentClient {
	return PaymentClient{
		clients: clients,
	}
}

func (c PaymentClient) PutRefundsWithResponse(ctx context.Context, command entity.RefundTicket) error {
	resp, err := c.clients.Payments.PutRefundsWithResponse(ctx, payments.PaymentRefundRequest{
		PaymentReference: command.TicketID,
		Reason:           "customer requested refund",
		DeduplicationId:  &command.Header.IdempotencyKey,
	})
	if err != nil {
		return err
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("unexpected status code while refunding payment: %d", resp.StatusCode())
	}

	return nil
}
