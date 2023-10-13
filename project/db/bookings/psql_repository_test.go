package bookings

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"tickets/db"
	"tickets/db/shows"
	"tickets/entity"
)

func TestPostgresRepository_Store(t *testing.T) {
	ctx := context.Background()
	container, url := db.StartPostgresContainer()
	defer container.Terminate(ctx)

	t.Setenv("POSTGRES_URL", url)
	db.GetDb(t)

	repo := NewPostgresRepository(db.GetDb(t))
	repoShows := shows.NewPostgresRepository(db.GetDb(t))

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
	errNoAvailableTickets := ErrNoAvailableTickets
	assert.ErrorAs(t, err, &errNoAvailableTickets)

}
