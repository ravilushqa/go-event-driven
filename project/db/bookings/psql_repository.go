package bookings

import (
	"context"

	"github.com/jmoiron/sqlx"

	"tickets/entity"
)

type PostgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Store(ctx context.Context, booking entity.Booking) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO bookings (booking_id, show_id, customer_email, number_of_tickets)
		VALUES (:booking_id, :show_id, :customer_email, :number_of_tickets)
		ON CONFLICT DO NOTHING -- ignore if already exists
	`, booking)
	return err
}
