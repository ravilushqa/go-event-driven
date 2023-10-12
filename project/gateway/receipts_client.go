package gateway

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/clients/receipts"

	"tickets/entity"
)

type ReceiptsClient struct {
	clients *clients.Clients
}

func NewReceiptsClient(clients *clients.Clients) ReceiptsClient {
	return ReceiptsClient{
		clients: clients,
	}
}

func (c ReceiptsClient) IssueReceipt(ctx context.Context, request entity.IssueReceiptRequest) (entity.IssueReceiptResponse, error) {
	body := receipts.PutReceiptsJSONRequestBody{
		TicketId: request.TicketID,
		Price: receipts.Money{
			MoneyAmount:   request.Price.Amount,
			MoneyCurrency: request.Price.Currency,
		},
		IdempotencyKey: &request.IdempotencyKey,
	}

	resp, err := c.clients.Receipts.PutReceiptsWithResponse(ctx, body)
	if err != nil {
		return entity.IssueReceiptResponse{}, err
	}

	switch resp.StatusCode() {
	case http.StatusOK:
		// receipt already exists
		return entity.IssueReceiptResponse{
			ReceiptNumber: resp.JSON200.Number,
			IssuedAt:      resp.JSON200.IssuedAt,
		}, nil
	case http.StatusCreated:
		// receipt was created
		return entity.IssueReceiptResponse{
			ReceiptNumber: resp.JSON201.Number,
			IssuedAt:      resp.JSON201.IssuedAt,
		}, nil
	default:
		return entity.IssueReceiptResponse{}, fmt.Errorf("unexpected status code for POST receipts-api/receipts: %d", resp.StatusCode())
	}
}
