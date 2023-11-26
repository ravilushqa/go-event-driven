package db

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"tickets/entity"
)

func TestPostgresRepository_Store(t *testing.T) {
	ctx := context.Background()
	container, url := StartPostgresContainer()
	defer container.Terminate(ctx)

	t.Setenv("POSTGRES_URL", url)
	GetDb(t)

	repo := NewBookingsPostgresRepository(GetDb(t))
	repoShows := NewShowsPostgresRepository(GetDb(t))

	show := entity.Show{
		ShowID:          uuid.NewString(),
		NumberOfTickets: 1,
	}
	err := repoShows.Store(ctx, show)
	assert.NoError(t, err)

	booking := entity.Booking{
		ShowID:          show.ShowID,
		NumberOfTickets: 2,
		CustomerEmail:   "test@test.io",
		BookingID:       uuid.NewString(),
	}

	err = repo.Store(ctx, booking, show.NumberOfTickets)
	errNoAvailableTickets := entity.ErrNoAvailableTickets
	assert.ErrorAs(t, err, &errNoAvailableTickets)

}
