package bookings

import (
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"tickets/entity"
	"tickets/pkg"
	"tickets/pkg/outbox"
)

type PostgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Store(ctx context.Context, booking entity.Booking) error {
	tx, err := r.db.Beginx()
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			rollbackErr := tx.Rollback()
			err = errors.Join(err, rollbackErr)
			return
		}
		err = tx.Commit()
	}()

	_, err = tx.NamedExecContext(ctx, `
		INSERT INTO 
		    bookings (booking_id, show_id, number_of_tickets, customer_email) 
		VALUES (:booking_id, :show_id, :number_of_tickets, :customer_email)
		`, booking)
	if err != nil {
		return fmt.Errorf("could not add booking: %w", err)
	}

	outboxPublisher, err := outbox.NewPublisherForDb(ctx, tx)
	if err != nil {
		return fmt.Errorf("could not create event bus: %w", err)
	}

	eventBus, err := pkg.NewEventBus(outboxPublisher)
	if err != nil {
		return err
	}

	err = eventBus.Publish(ctx, entity.BookingMade{
		Header:          entity.NewEventHeader(),
		BookingID:       booking.BookingID,
		NumberOfTickets: booking.NumberOfTickets,
		CustomerEmail:   booking.CustomerEmail,
		ShowID:          booking.ShowID,
	})
	if err != nil {
		return fmt.Errorf("could not publish event: %w", err)
	}

	return nil
}
