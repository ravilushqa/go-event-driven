package read_model_ops_bookings

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"tickets/db"
	"tickets/entity"
)

func TestPostgresRepository_Store(t *testing.T) {
	ctx := context.Background()
	container, url := db.StartPostgresContainer()
	defer container.Terminate(ctx)

	t.Setenv("POSTGRES_URL", url)
	dbConn := db.GetDb(t)

	repo := NewPostgresRepository(dbConn)

	t.Run("without tickets", func(t *testing.T) {
		model := entity.OpsBooking{
			BookingID:  uuid.NewString(),
			BookedAt:   time.Now(),
			Tickets:    nil,
			LastUpdate: time.Now(),
		}

		err := repo.Store(ctx, model)
		assert.NoError(t, err)

		// Test idempotency
		err = repo.Store(ctx, model)
		assert.NoError(t, err)

		tx, err := dbConn.Beginx()
		assert.NoError(t, err)
		booking, err := repo.GetByID(ctx, tx, model.BookingID)
		assert.NoError(t, err)

		err = tx.Commit()
		assert.NoError(t, err)

		assert.Equal(t, model.BookingID, booking.BookingID)
		//assert.Equal(t, model.BookedAt.Unix(), booking.BookedAt.Unix())
		assert.Equal(t, model.LastUpdate.Unix(), booking.LastUpdate.Unix())
		assert.Equal(t, model.Tickets, booking.Tickets)
	})

	t.Run("with tickets", func(t *testing.T) {
		model := entity.OpsBooking{
			BookingID: uuid.NewString(),
			BookedAt:  time.Now(),
			Tickets: map[string]entity.OpsTicket{
				uuid.NewString(): {ReceiptNumber: "123"},
				uuid.NewString(): {ReceiptNumber: "456"},
			},
			LastUpdate: time.Now(),
		}

		err := repo.Store(ctx, model)
		assert.NoError(t, err)

		// Test idempotency
		err = repo.Store(ctx, model)
		assert.NoError(t, err)

		tx, err := dbConn.Beginx()
		assert.NoError(t, err)
		booking, err := repo.GetByID(ctx, tx, model.BookingID)
		assert.NoError(t, err)

		err = tx.Commit()
		assert.NoError(t, err)

		assert.Equal(t, model.BookingID, booking.BookingID)
		//assert.Equal(t, model.BookedAt.Unix(), booking.BookedAt.Unix())
		assert.Equal(t, model.LastUpdate.Unix(), booking.LastUpdate.Unix())
		assert.Equal(t, model.Tickets, booking.Tickets)
	})
}
