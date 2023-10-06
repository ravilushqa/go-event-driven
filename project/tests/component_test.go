package tests_test

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

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/lithammer/shortuuid/v3"
	redis2 "github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
	"golang.org/x/sync/errgroup"

	"tickets/entity"
	httpHandler "tickets/handler/http"
	"tickets/handler/pubsub"
	"tickets/mocks"
	"tickets/pkg"
)

const (
	httpAddress  = ":8080"
	redisAddress = "localhost:6379"
)

func TestComponent(t *testing.T) {
	defer goleak.VerifyNone(t)
	receiptsClient := mocks.NewMockReceiptsService(t)
	receiptsClient.IssueReceiptFunc = func(ctx context.Context, request entity.IssueReceiptRequest) (entity.IssueReceiptResponse, error) {
		return entity.IssueReceiptResponse{
			ReceiptNumber: "receipt-1",
			IssuedAt:      time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
		}, nil
	}
	spreadsheetsClient := mocks.NewMockSpreadsheetsAPI(t)

	done := make(chan struct{})
	go func() {
		<-done
		e := syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
		require.NoError(t, e)
	}()

	finished := make(chan struct{})
	go func() {
		err := startServer(t, receiptsClient, spreadsheetsClient)
		assert.NoError(t, err)
		close(finished)
	}()

	defer func() {
		close(done)
		<-finished
	}()

	waitForHttpServer(t)

	sendTicketsStatus(t, TicketsStatusRequest{
		Tickets: []TicketStatus{
			{
				TicketID:  "ticket-1",
				Status:    "confirmed",
				Price:     Money{Amount: "100", Currency: "USD"},
				Email:     "test@test.io",
				BookingID: "booking-1",
			},
		},
	})

	assertReceiptForTicketIssued(t, receiptsClient, TicketStatus{
		TicketID:  "ticket-1",
		Status:    "confirmed",
		Price:     Money{Amount: "100", Currency: "USD"},
		Email:     "test@test.io",
		BookingID: "booking-1",
	})
}

func startServer(t *testing.T, receiptsClient *mocks.MockReceiptsService, spreadsheetsClient *mocks.MockSpreadsheetsAPI) error {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	log.Init(logrus.InfoLevel)

	redisClient := pkg.NewRedisClient(redisAddress)
	defer func(redisClient *redis2.Client) {
		err := redisClient.Close()
		assert.NoError(t, err)
	}(redisClient)

	watermillLogger := log.NewWatermill(logrus.NewEntry(logrus.StandardLogger()))

	redisPublisher := pkg.NewRedisPublisher(redisClient, watermillLogger)

	watermillRouter := pubsub.NewWatermillRouter(receiptsClient, spreadsheetsClient, redisClient, watermillLogger)

	httpServer := httpHandler.NewServer(redisPublisher, spreadsheetsClient, httpAddress)

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return watermillRouter.Run(ctx)
	})

	g.Go(func() error {
		// we don't want to start HTTP server before Watermill router (so service won't be healthy before it's ready)
		<-watermillRouter.Running()

		err := httpServer.Run(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	return g.Wait()
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
