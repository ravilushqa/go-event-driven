package db

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tickets/db/tickets"
	"tickets/entity"
)

var db *sqlx.DB
var getDbOnce sync.Once

func getDb(t *testing.T) *sqlx.DB {
	getDbOnce.Do(func() {
		var err error
		db, err = sqlx.Open("postgres", os.Getenv("POSTGRES_URL"))
		assert.NoError(t, err)
		t.Cleanup(func() {
			db.Close()
		})

		err = InitializeDatabaseSchema(db)
		assert.NoError(t, err)
	})
	return db
}

func TestTicketsRepository_Add_idempotency(t *testing.T) {
	ctx := context.Background()

	db = getDb(t)

	err := InitializeDatabaseSchema(db)
	require.NoError(t, err)

	repo := tickets.NewPostgresRepository(db)

	ticketToAdd := entity.Ticket{
		TicketID:      uuid.NewString(),
		PriceAmount:   "30.00",
		PriceCurrency: "EUR",
		CustomerEmail: "foo@bar.com",
	}

	for i := 0; i < 2; i++ {
		err = repo.Store(ctx, ticketToAdd)
		require.NoError(t, err)

		// probably it would be good to have a method to get ticket by ID
		list, err := repo.FindAll(ctx)
		require.NoError(t, err)

		// add should be idempotent, so the method should always return 1
		require.Len(t, list, 1)
	}
}
