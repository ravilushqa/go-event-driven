package mocks

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// MockSpreadsheetsAPI implements SpreadsheetsAPI for testing purposes
type MockSpreadsheetsAPI struct {
	mu            sync.Mutex
	t             *testing.T
	AppendRowFunc func(ctx context.Context, sheetName string, row []string) error
	AppendedRows  map[string][][]string
}

// NewMockSpreadsheetsAPI creates a new mock for SpreadsheetsAPI
func NewMockSpreadsheetsAPI(t *testing.T) *MockSpreadsheetsAPI {
	if t == nil {
		panic("missing required argument 't'")
	}

	return &MockSpreadsheetsAPI{t: t, AppendedRows: make(map[string][][]string)}
}

// AppendRow mock implementation
func (m *MockSpreadsheetsAPI) AppendRow(ctx context.Context, sheetName string, row []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.AppendedRows[sheetName] = append(m.AppendedRows[sheetName], row)

	if m.AppendRowFunc != nil {
		return m.AppendRowFunc(ctx, sheetName, row)
	}
	return errors.New("AppendRow not implemented")
}
