package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"

	"tickets/entity"
)

func TestTicketsRepository_Add_idempotency(t *testing.T) {
	ctx := context.Background()
	container, url := StartPostgresContainer()
	defer container.Terminate(ctx)

	t.Setenv("POSTGRES_URL", url)
	db := GetDb(t)
	repo := NewTicketsPostgresRepository(db)

	ticketToAdd := entity.Ticket{
		TicketID:      uuid.NewString(),
		PriceAmount:   "30.00",
		PriceCurrency: "EUR",
		CustomerEmail: "foo@bar.com",
	}

	for i := 0; i < 2; i++ {
		err := repo.Store(ctx, ticketToAdd)
		require.NoError(t, err)

		// probably it would be good to have a method to get ticket by ID
		list, err := repo.FindAll(ctx)
		require.NoError(t, err)

		// add should be idempotent, so the method should always return 1
		require.Len(t, list, 1)
	}
}
