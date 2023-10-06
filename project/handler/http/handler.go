package http

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

type Handler struct {
	publisher             message.Publisher
	spreadsheetsAPIClient SpreadsheetsAPI
}

type SpreadsheetsAPI interface {
	AppendRow(ctx context.Context, spreadsheetName string, row []string) error
}
