package gateway

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ThreeDotsLabs/go-event-driven/common/clients"
	"github.com/ThreeDotsLabs/go-event-driven/common/log"
)

type FilesClient struct {
	clients *clients.Clients
}

func NewFilesClient(clients *clients.Clients) FilesClient {
	return FilesClient{
		clients: clients,
	}
}

func (c FilesClient) UploadFile(ctx context.Context, fileID string, fileContent string) error {
	resp, err := c.clients.Files.PutFilesFileIdContentWithTextBodyWithResponse(ctx, fileID, fileContent)
	if err != nil {
		return fmt.Errorf("failed to upload file %s: %w", fileID, err)
	}

	if resp.StatusCode() == http.StatusConflict {
		log.FromContext(ctx).Infof("file %s already exists", fileID)
		return nil
	}
	if resp.StatusCode() != http.StatusCreated {
		return fmt.Errorf("unexpected status code while uploading file %s: %d", fileID, resp.StatusCode())
	}

	return nil
}

func (c FilesClient) DownloadFile(ctx context.Context, fileID string) (string, error) {
	resp, err := c.clients.Files.GetFilesFileIdContentWithResponse(ctx, fileID)
	if err != nil {
		return "", fmt.Errorf("get file content: %w", err)
	}

	if resp.StatusCode() == http.StatusNotFound {
		return "", nil
	}
	if resp.StatusCode() != http.StatusOK {
		return "", fmt.Errorf("unexpected status code while getting file %s: %d", fileID, resp.StatusCode())
	}

	return string(resp.Body), nil
}
