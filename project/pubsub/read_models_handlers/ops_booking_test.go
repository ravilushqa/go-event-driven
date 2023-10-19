package read_models_handlers

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"tickets/db"
	"tickets/db/read_model_ops_bookings"
	"tickets/entity"
)

func TestOpsBookingReadModel(t *testing.T) {
	ctx := context.Background()
	container, url := db.StartPostgresContainer()
	defer container.Terminate(ctx)

	t.Setenv("POSTGRES_URL", url)
	dbConn := db.GetDb(t)

	repo := read_model_ops_bookings.NewPostgresRepository(dbConn)
	readModel := NewOpsBookingReadModel(repo)

	bookingID := uuid.NewString()
	ticketID := uuid.NewString()

	t.Run("bookingMade", func(t *testing.T) {
		bookingMade := &entity.BookingMade{
			Header:        entity.NewEventHeader(),
			BookingID:     bookingID,
			CustomerEmail: "test@test.test",
		}
		assert.NoError(t, readModel.OnBookingMade(ctx, bookingMade))
	})

	t.Run("ticketBookingConfirmed", func(t *testing.T) {
		ticketReceiptIssued := &entity.TicketBookingConfirmed{
			Header:        entity.NewEventHeader(),
			TicketID:      ticketID,
			CustomerEmail: "test@test.test",
			Price:         entity.Money{Amount: "100", Currency: "EUR"},
			BookingID:     bookingID,
		}
		assert.NoError(t, readModel.OnTicketBookingConfirmed(ctx, ticketReceiptIssued))

		booking, err := repo.Get(ctx, ticketReceiptIssued.BookingID)
		assert.NoError(t, err)

		assert.Equal(t, 1, len(booking.Tickets))
	})

	t.Run("ticketReceiptIssued", func(t *testing.T) {
		ticketReceiptIssued := &entity.TicketReceiptIssued{
			Header:        entity.NewEventHeader(),
			TicketID:      ticketID,
			ReceiptNumber: "123",
		}
		assert.NoError(t, readModel.OnTicketReceiptIssued(ctx, ticketReceiptIssued))

		tx := dbConn.MustBegin()
		booking, err := repo.GetByTicketID(ctx, tx, ticketReceiptIssued.TicketID)
		if err != nil {
			return
		}
		tx.Commit()

		assert.Equal(t, 1, len(booking.Tickets))
		assert.Equal(t, ticketReceiptIssued.ReceiptNumber, booking.Tickets[ticketReceiptIssued.TicketID].ReceiptNumber)
	})
}
