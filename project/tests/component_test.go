package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/lithammer/shortuuid/v3"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.uber.org/goleak"

	"tickets/app"
	"tickets/db"
	"tickets/entity"
	"tickets/gateway"
)

var httpAddress = ":8080"

func TestComponent(t *testing.T) {
	defer goleak.VerifyNone(t,
		// used for testcontainers
		goleak.IgnoreTopFunction("github.com/testcontainers/testcontainers-go.(*Reaper).Connect.func1"),
		// used for test http queries
		goleak.IgnoreTopFunction("net/http.(*persistConn).readLoop"),
		// used for test http queries
		goleak.IgnoreTopFunction("net/http.(*persistConn).writeLoop"),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dbconn, err := sqlx.Open("postgres", postgresURL)
	if err != nil {
		panic(err)
	}
	defer dbconn.Close()

	redisClient := redis.NewClient(&redis.Options{Addr: redisURL})
	defer redisClient.Close()

	spreadsheetsClient := &gateway.SpreadsheetsMock{}
	receiptsClient := &gateway.ReceiptsMock{}
	filesClient := &gateway.FilesMock{}
	deadNationClient := &gateway.DeadNationMock{}
	paymentClient := &gateway.PaymentMock{}
	transClient := &gateway.TransportationMock{}
	traceProvider := trace.NewTracerProvider()

	vbRepo := db.NewVipBundlePostgresRepository(dbconn)

	go func() {
		svc := app.New(
			httpAddress,
			dbconn,
			redisClient,
			spreadsheetsClient,
			receiptsClient,
			filesClient,
			deadNationClient,
			paymentClient,
			transClient,
			traceProvider,
		)
		require.NoError(t, svc.Run(ctx))
	}()

	waitForHttpServer(t)

	ticket := TicketStatus{
		TicketID:  uuid.NewString(),
		Status:    "confirmed",
		Price:     Money{Amount: "100", Currency: "USD"},
		Email:     "test@test.io",
		BookingID: uuid.NewString(),
	}

	idempotencyKey := uuid.NewString()

	// check idempotency
	for i := 0; i < 3; i++ {
		sendTicketsStatus(t, TicketsStatusRequest{Tickets: []TicketStatus{ticket}}, idempotencyKey)
	}

	assertReceiptForTicketIssued(t, receiptsClient, ticket)
	assertTicketPrinted(t, filesClient, ticket)
	assertRowToSheetAdded(t, spreadsheetsClient, ticket, "tickets-to-print")
	assertTicketStoredInRepository(t, dbconn, ticket)

	sendTicketsStatus(t, TicketsStatusRequest{Tickets: []TicketStatus{
		{
			TicketID: ticket.TicketID,
			Status:   "canceled",
			Email:    ticket.Email,
		},
	}}, uuid.NewString())

	assertRowToSheetAdded(t, spreadsheetsClient, ticket, "tickets-to-refund")
	showID := sendPostShow(t, postShowsRequest{
		DeadNationID:    uuid.NewString(),
		NumberOfTickets: 5,
		StartTime:       time.Now().Add(time.Hour),
		Title:           "test",
		Venue:           "test",
	})

	bookResp := bookTickets(t, postBookTicketsRequest{
		ShowID:          showID,
		NumberOfTickets: 3,
		CustomerEmail:   "test@test.io",
	})
	require.Equal(t, http.StatusCreated, bookResp.StatusCode)

	bookingID := postBookTicketsResponse{}
	err = json.NewDecoder(bookResp.Body).Decode(&bookingID)
	require.NoError(t, err)

	// overbooking
	bookResp = bookTickets(t, postBookTicketsRequest{
		ShowID:          showID,
		NumberOfTickets: 3,
		CustomerEmail:   "test@test.io",
	})
	require.Equal(t, http.StatusBadRequest, bookResp.StatusCode)

	// refund
	resp := sentTicketRefund(t, bookingID.BookingID)
	require.Equal(t, http.StatusAccepted, resp.StatusCode)
	assertVoidReceipt(t, receiptsClient, ticket.TicketID)
	assertRefundIssued(t, paymentClient, ticket.TicketID)

	// vip bundle
	vb := vipBundleRequest{
		CustomerEmail:   "test1@test.io",
		InboundFlightID: uuid.NewString(),
		NumberOfTickets: 1,
		Passengers:      []string{"test1"},
		ReturnFlightID:  uuid.NewString(),
		ShowID:          showID,
	}

	resp = sendBookVipBundle(t, vb)
	var vbResp vipBundleResponse
	err = json.NewDecoder(resp.Body).Decode(&vbResp)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	assertVipBundleSuccessfullyBooked(t, vbRepo, vbResp)
}

func assertVipBundleSuccessfullyBooked(t *testing.T, vipBundleRepo entity.VipBundleRepository, resp vipBundleResponse) {
	require.Eventually(
		t,
		func() bool {
			vb, err := vipBundleRepo.Get(context.Background(), resp.VipBundleID)
			return err == nil &&
				resp.VipBundleID == vb.VipBundleID &&
				resp.BookingID == vb.BookingID &&
				vb.IsFinalized &&
				!vb.Failed
		},
		10*time.Second,
		100*time.Millisecond,
	)
}

func assertVoidReceipt(t *testing.T, client *gateway.ReceiptsMock, id string) {
	require.Eventually(
		t,
		func() bool {
			return len(client.VoidedReceipts) == 1
		},
		10*time.Second,
		100*time.Millisecond,
		"receipt for booking %s not voided", id,
	)
}

func assertRefundIssued(t *testing.T, client *gateway.PaymentMock, id string) {
	require.Eventually(
		t,
		func() bool {
			return len(client.Refunds) == 1
		},
		10*time.Second,
		100*time.Millisecond,
		"refund for booking %s not issued", id,
	)
}

func assertTicketStoredInRepository(t *testing.T, sqlxDB *sqlx.DB, ticket TicketStatus) {
	ticketsRepo := db.NewTicketsPostgresRepository(sqlxDB)

	require.Eventually(
		t,
		func() bool {
			all, err := ticketsRepo.FindAll(context.Background())
			if err != nil {
				return false
			}

			for _, t := range all {
				if t.TicketID == ticket.TicketID {
					return true
				}
			}

			return false
		},
		10*time.Second,
		100*time.Millisecond,
	)
}

func assertRowToSheetAdded(t *testing.T, spreadsheetsService *gateway.SpreadsheetsMock, ticket TicketStatus, sheetName string) {
	require.Eventually(
		t,
		func() bool {
			rows, ok := spreadsheetsService.Rows[sheetName]
			require.True(t, ok, "sheet %s not found", sheetName)

			var allValues []string
			for _, row := range rows {
				allValues = append(allValues, row...)
			}

			require.Contains(t, allValues, ticket.TicketID, "ticket id not found in sheet %s", sheetName)
			return true
		},
		10*time.Second,
		100*time.Millisecond,
	)
}

func assertTicketPrinted(t *testing.T, filesAPI *gateway.FilesMock, ticket TicketStatus) {
	require.Eventually(
		t,
		func() bool {
			content, err := filesAPI.DownloadFile(context.Background(), ticket.TicketID+"-ticket.html")
			require.NoError(t, err)
			require.NotEmpty(t, content)
			require.Contains(t, content, ticket.TicketID)

			return true
		},
		10*time.Second,
		100*time.Millisecond,
	)
}

func assertReceiptForTicketIssued(t *testing.T, receiptsService *gateway.ReceiptsMock, ticket TicketStatus) {
	require.Eventually(
		t,
		func() bool {
			return len(receiptsService.IssuedReceipts) == 1
		},
		10*time.Second,
		100*time.Millisecond,
		"receipt for ticket %s not found", ticket.TicketID,
	)

	receipt, ok := lo.Find(lo.Values(receiptsService.IssuedReceipts), func(r entity.IssueReceiptRequest) bool {
		return r.TicketID == ticket.TicketID
	})
	require.Truef(t, ok, "receipt for ticket %s not found", ticket.TicketID)
	require.Equal(t, ticket.TicketID, receipt.TicketID)
	require.Equal(t, ticket.Price.Amount, receipt.Price.Amount)
	require.Equal(t, ticket.Price.Currency, receipt.Price.Currency)
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

type postShowsRequest struct {
	DeadNationID    string    `json:"dead_nation_id"`
	NumberOfTickets int       `json:"number_of_tickets"`
	StartTime       time.Time `json:"start_time"`
	Title           string    `json:"title"`
	Venue           string    `json:"venue"`
}

type postShowsResponse struct {
	ShowID string `json:"show_id"`
}

type postBookTicketsRequest struct {
	ShowID          string `json:"show_id"`
	NumberOfTickets int    `json:"number_of_tickets"`
	CustomerEmail   string `json:"customer_email"`
}

type postBookTicketsResponse struct {
	BookingID string `json:"booking_id"`
}

type vipBundleRequest struct {
	CustomerEmail   string   `json:"customer_email"`
	InboundFlightID string   `json:"inbound_flight_id"`
	NumberOfTickets int      `json:"number_of_tickets"`
	Passengers      []string `json:"passengers"`
	ReturnFlightID  string   `json:"return_flight_id"`
	ShowID          string   `json:"show_id"`
}

type vipBundleResponse struct {
	BookingID   string `json:"booking_id"`
	VipBundleID string `json:"vip_bundle_id"`
}

func sendTicketsStatus(t *testing.T, req TicketsStatusRequest, idempotencyKey string) {
	t.Helper()

	payload, err := json.Marshal(req)
	require.NoError(t, err)

	correlationID := shortuuid.New()

	httpReq, err := http.NewRequest(
		http.MethodPost,
		"http://localhost:8080/tickets-status",
		bytes.NewBuffer(payload),
	)
	require.NoError(t, err)

	httpReq.Header.Set("Correlation-ID", correlationID)
	httpReq.Header.Set("Idempotency-Key", idempotencyKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func sendPostShow(t *testing.T, request postShowsRequest) string {
	t.Helper()

	correlationID := shortuuid.New()

	payload, err := json.Marshal(request)
	require.NoError(t, err)
	httpReq, err := http.NewRequest(
		http.MethodPost,
		"http://localhost:8080/shows",
		bytes.NewBuffer(payload),
	)
	httpReq.Header.Set("Correlation-ID", correlationID)
	httpReq.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var response postShowsResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	return response.ShowID
}

func sentTicketRefund(t *testing.T, bookingID string) *http.Response {
	t.Helper()

	correlationID := shortuuid.New()

	httpReq, err := http.NewRequest(
		http.MethodPut,
		"http://localhost:8080/ticket-refund/"+bookingID,
		nil,
	)
	httpReq.Header.Set("Correlation-ID", correlationID)
	httpReq.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)

	return resp
}

func bookTickets(t *testing.T, request postBookTicketsRequest) *http.Response {
	t.Helper()

	payload, err := json.Marshal(request)
	require.NoError(t, err)

	httpReq, err := http.NewRequest(
		http.MethodPost,
		"http://localhost:8080/book-tickets",
		bytes.NewBuffer(payload),
	)
	httpReq.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)
	return resp
}

func sendBookVipBundle(t *testing.T, vb vipBundleRequest) *http.Response {
	t.Helper()

	payload, err := json.Marshal(vb)
	require.NoError(t, err)

	httpReq, err := http.NewRequest(
		http.MethodPost,
		"http://localhost:8080/book-vip-bundle",
		bytes.NewBuffer(payload),
	)
	httpReq.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(httpReq)
	require.NoError(t, err)
	return resp
}

func waitForHttpServer(t *testing.T) {
	t.Helper()

	require.Eventually(
		t,
		func() bool {
			resp, err := http.Get("http://localhost:8080/health")
			if err != nil {
				return false
			}
			defer resp.Body.Close()
			return resp.StatusCode < 300
		},
		time.Second*10,
		time.Millisecond*50,
		"API not ready",
	)
}
