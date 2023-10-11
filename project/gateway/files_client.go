package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"

	"tickets/entity"
)

type FilesClient struct {
	clients *clients.Clients
}

func NewFilesClient(clients *clients.Clients) FilesClient {
	return FilesClient{
		clients: clients,
	}
}

func (c FilesClient) Put(ctx context.Context, ticket entity.TicketBookingConfirmed) (string, error) {
	name := fmt.Sprintf("%s-ticket.html", ticket.TicketID)
	body, err := json.Marshal(ticket)
	if err != nil {
		return "", err
	}

	resp, err := c.clients.Files.PutFilesFileIdContentWithTextBodyWithResponse(ctx, name, string(body))
	if err != nil {
		return "", err
	}

	if resp.StatusCode() == http.StatusConflict {
		log.FromContext(ctx).Infof("file %s already exists", name)
		return name, nil
	}

	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("unexpected status code for PUT files-api/files/%s/content: %d", name, resp.StatusCode())
	}

	return name, nil
}
