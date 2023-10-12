package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/lithammer/shortuuid/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"tickets/entity"
	"tickets/mocks"
	"tickets/pkg"
	"tickets/service"
)

var (
	httpAddress = ":8080"
)

func TestComponent(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreTopFunction("github.com/testcontainers/testcontainers-go.(*Reaper).Connect.func1"))
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	dbconn, err := sqlx.Open("postgres", postgresURL)
	if err != nil {
		panic(err)
	}
	defer dbconn.Close()

	redisClient := pkg.NewRedisClient(redisURL)
	defer redisClient.Close()

	receiptsClient := mocks.NewMockReceiptsService(t)
	receiptsClient.IssueReceiptFunc = func(ctx context.Context, request entity.IssueReceiptRequest) (entity.IssueReceiptResponse, error) {
		return entity.IssueReceiptResponse{
			ReceiptNumber: "receipt-1",
			IssuedAt:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		}, nil
	}
	spreadsheetsClient := mocks.NewMockSpreadsheetsAPI(t)
	spreadsheetsClient.AppendRowFunc = func(ctx context.Context, spreadsheetID string, ticketIDs []string) error {
		return nil
	}

	filesClient := mocks.NewFileService(t)
	filesClient.On("Put", mock.Anything, mock.Anything).Return("file-id", nil)

	done := make(chan struct{})
	go func() {
		<-done
		e := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		require.NoError(t, e)
	}()

	finished := make(chan struct{})
	go func() {
		svc := service.New(
			dbconn,
			redisClient,
			spreadsheetsClient,
			receiptsClient,
			filesClient,
			httpAddress,
		)
		assert.NoError(t, svc.Run(ctx))
		close(finished)
	}()

	defer func() {
		close(done)
		<-finished
	}()

	waitForHttpServer(t)

	ticket := TicketStatus{
		TicketID:  uuid.NewString(),
		Status:    "confirmed",
		Price:     Money{Amount: "100", Currency: "USD"},
		Email:     "test@test.io",
		BookingID: "booking-1",
	}

	sendTicketsStatus(t, TicketsStatusRequest{Tickets: []TicketStatus{ticket}})

	sendTicketsStatus(t, TicketsStatusRequest{Tickets: []TicketStatus{
		{
			TicketID: ticket.TicketID,
			Status:   "canceled",
			Email:    ticket.Email,
		},
	}})

	sendTicketsStatus(t, TicketsStatusRequest{Tickets: []TicketStatus{
		{
			TicketID: ticket.TicketID,
			Status:   "canceled",
			Email:    ticket.Email,
		},
	}})

	assertRowToSheetAdded(t, spreadsheetsClient, ticket, "tickets-to-refund")
	assertReceiptForTicketIssued(t, receiptsClient, ticket)
}

func waitForHttpServer(t *testing.T) {
	t.Helper()

	require.EventuallyWithT(
		t,
		func(t *assert.CollectT) {
			resp, err := http.Get("http://localhost:8080/health")
			if !assert.NoError(t, err) {
				return
			}
			defer resp.Body.Close()

			if assert.Less(t, resp.StatusCode, 300, "API not ready, http status: %d", resp.StatusCode) {
				return
			}
		},
		time.Second*10,
		time.Millisecond*50,
	)
}

type TicketsStatusRequest struct {
	Tickets []TicketStatus `json:"tickets"`
}

type TicketStatus struct {
	TicketID  string `json:"ticket_id"`
	Status    string `json:"status"`
	Price     Money  `json:"price"`
	Email     string `json:"email"`
	BookingID string `json:"booking_id"`
}

type Money struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

func sendTicketsStatus(t *testing.T, req TicketsStatusRequest) {
	t.Helper()

	payload, err := json.Marshal(req)
	require.NoError(t, err)

	correlationID := shortuuid.New()

	ticketIDs := make([]string, 0, len(req.Tickets))
	for _, ticket := range req.Tickets {
		ticketIDs = append(ticketIDs, ticket.TicketID)
	}

	httpReq, err := http.NewRequest(
		http.MethodPost,
		"http://localhost:8080/tickets-status",
		bytes.NewBuffer(payload),
	)
	require.NoError(t, err)

	httpReq.Header.Set("Correlation-ID", correlationID)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func assertReceiptForTicketIssued(t *testing.T, receiptsService *mocks.MockReceiptsService, ticket TicketStatus) {
	assert.EventuallyWithT(
		t,
		func(collectT *assert.CollectT) {
			issuedReceipts := len(receiptsService.IssuedReceipts)
			t.Log("issued receipts", issuedReceipts)

			assert.Greater(collectT, issuedReceipts, 0, "no receipts issued")
		},
		10*time.Second,
		100*time.Millisecond,
	)

	var receipt entity.IssueReceiptRequest
	var ok bool
	for _, issuedReceipt := range receiptsService.IssuedReceipts {
		if issuedReceipt.TicketID != ticket.TicketID {
			continue
		}
		receipt = issuedReceipt
		ok = true
		break
	}
	require.Truef(t, ok, "receipt for ticket %s not found", ticket.TicketID)

	assert.Equal(t, ticket.TicketID, receipt.TicketID)
	assert.Equal(t, ticket.Price.Amount, receipt.Price.Amount)
	assert.Equal(t, ticket.Price.Currency, receipt.Price.Currency)
}

func assertRowToSheetAdded(t *testing.T, spreadsheetsService *mocks.MockSpreadsheetsAPI, ticket TicketStatus, sheetName string) bool {
	return assert.EventuallyWithT(
		t,
		func(t *assert.CollectT) {
			rows, ok := spreadsheetsService.AppendedRows[sheetName]
			if !assert.True(t, ok, "sheet %s not found", sheetName) {
				return
			}

			var allValues []string

			for _, row := range rows {
				for _, col := range row {
					allValues = append(allValues, col)
				}
			}

			assert.Contains(t, allValues, ticket.TicketID, "ticket id not found in sheet %s", sheetName)
		},
		10*time.Second,
		100*time.Millisecond,
	)
}
