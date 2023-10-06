package mocks

import (
	"context"
	"errors"
	"testing"

	"tickets/entity"
)

// MockReceiptsService implements ReceiptsService for testing purposes
type MockReceiptsService struct {
	t                *testing.T
	IssueReceiptFunc func(ctx context.Context, request *entity.IssueReceiptRequest) error
}

// NewMockReceiptsService creates a new mock for ReceiptsService
func NewMockReceiptsService(t *testing.T) *MockReceiptsService {
	if t == nil {
		panic("missing required argument 't'")
	}

	return &MockReceiptsService{t: t}
}

// IssueReceipt mock implementation
func (m *MockReceiptsService) IssueReceipt(ctx context.Context, request *entity.IssueReceiptRequest) error {
	if m.IssueReceiptFunc != nil {
		return m.IssueReceiptFunc(ctx, request)
	}
	return errors.New("IssueReceipt not implemented")
}
