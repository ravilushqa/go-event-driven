package mocks

import (
	"context"
	"errors"
	"testing"
)

// MockSpreadsheetsAPI implements SpreadsheetsAPI for testing purposes
type MockSpreadsheetsAPI struct {
	t             *testing.T
	AppendRowFunc func(ctx context.Context, sheetName string, row []string) error
}

// NewMockSpreadsheetsAPI creates a new mock for SpreadsheetsAPI
func NewMockSpreadsheetsAPI(t *testing.T) *MockSpreadsheetsAPI {
	if t == nil {
		panic("missing required argument 't'")
	}

	return &MockSpreadsheetsAPI{t: t}
}

// AppendRow mock implementation
func (m *MockSpreadsheetsAPI) AppendRow(ctx context.Context, sheetName string, row []string) error {
	if m.AppendRowFunc != nil {
		return m.AppendRowFunc(ctx, sheetName, row)
	}
	return errors.New("AppendRow not implemented")
}
