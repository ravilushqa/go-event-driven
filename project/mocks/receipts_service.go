package mocks

import (
	"context"
	"errors"
	"sync"
	"testing"

	"tickets/entity"
)

// MockReceiptsService implements ReceiptsService for testing purposes
type MockReceiptsService struct {
	mu               sync.Mutex
	t                *testing.T
	IssueReceiptFunc func(ctx context.Context, request entity.IssueReceiptRequest) (entity.IssueReceiptResponse, error)
	IssuedReceipts   []entity.IssueReceiptRequest
}

// NewMockReceiptsService creates a new mock for ReceiptsService
func NewMockReceiptsService(t *testing.T) *MockReceiptsService {
	if t == nil {
		panic("missing required argument 't'")
	}

	return &MockReceiptsService{t: t, IssuedReceipts: make([]entity.IssueReceiptRequest, 0)}
}

// IssueReceipt mock implementation
func (m *MockReceiptsService) IssueReceipt(ctx context.Context, request entity.IssueReceiptRequest) (entity.IssueReceiptResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.IssuedReceipts = append(m.IssuedReceipts, request)
	if m.IssueReceiptFunc == nil {
		return entity.IssueReceiptResponse{}, errors.New("IssueReceipt not implemented")
	}

	return m.IssueReceiptFunc(ctx, request)
}
