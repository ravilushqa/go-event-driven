package bookings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"tickets/entity"
	"tickets/pkg"
	"tickets/pkg/outbox"
)

var ErrNoAvailableTickets = errors.New("no available tickets")

type PostgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

// Store stores booking in database and publishes event to event bus.
// It uses transaction to ensure that booking is stored only if there are available tickets.
func (r *PostgresRepository) Store(ctx context.Context, booking entity.Booking, showTicketsCount int) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
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

	availableTickets, err := r.getAvailableTickets(ctx, tx, booking.ShowID, showTicketsCount)
	if err != nil {
		return fmt.Errorf("could not get available tickets: %w", err)
	}

	if availableTickets < booking.NumberOfTickets {
		return ErrNoAvailableTickets
	}

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

func (r *PostgresRepository) getAvailableTickets(ctx context.Context, tx *sqlx.Tx, showID string, showTicketsCount int) (int, error) {
	var bookedTicketsCount int
	err := tx.GetContext(ctx, &bookedTicketsCount, `
		SELECT 
		    COALESCE(SUM(number_of_tickets), 0) 
		FROM 
		    bookings 
		WHERE 
		    show_id = $1
		GROUP BY 
		    show_id
		`, showID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return showTicketsCount, nil
		}
		return 0, fmt.Errorf("could not get booked tickets count: %w", err)
	}

	return showTicketsCount - bookedTicketsCount, nil
}
