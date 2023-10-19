package read_model_ops_bookings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"tickets/entity"
)

type PostgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r PostgresRepository) FindAll(ctx context.Context) ([]entity.OpsBooking, error) {
	var bookingsData [][]byte
	err := r.db.SelectContext(ctx, &bookingsData, `
		SELECT payload 
		FROM read_model_ops_bookings
	`)
	if err != nil {
		return nil, fmt.Errorf("could not get booking read models: %w", err)
	}

	var bookings []entity.OpsBooking
	for _, bookingData := range bookingsData {
		var booking entity.OpsBooking
		if err = json.Unmarshal(bookingData, &booking); err != nil {
			return nil, fmt.Errorf("could not unmarshal booking read model: %w", err)
		}
		bookings = append(bookings, booking)
	}

	return bookings, nil
}

func (r PostgresRepository) Get(ctx context.Context, bookingID string) (entity.OpsBooking, error) {
	var payload []byte
	err := r.db.GetContext(ctx, &payload, `
		SELECT payload 
		FROM read_model_ops_bookings 
		WHERE booking_id = $1
		`, bookingID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.OpsBooking{}, nil
		}
		return entity.OpsBooking{}, fmt.Errorf("could not get booking read model: %w", err)
	}

	var booking entity.OpsBooking
	if err = json.Unmarshal(payload, &booking); err != nil {
		return entity.OpsBooking{}, fmt.Errorf("could not unmarshal booking read model: %w", err)
	}

	return booking, nil
}

// Store stores booking in database. Idempotent. On conflict do nothing.
func (r PostgresRepository) Store(ctx context.Context, booking entity.OpsBooking) error {
	payload, err := json.Marshal(booking)
	if err != nil {
		return err
	}
	_, err = r.db.NamedExecContext(ctx, `
		INSERT INTO 
		    read_model_ops_bookings (booking_id, payload) 
		VALUES (:booking_id, :payload)
		ON CONFLICT (booking_id) DO NOTHING
		`, map[string]interface{}{
		"booking_id": booking.BookingID,
		"payload":    payload,
	})
	if err != nil {
		return fmt.Errorf("could not add booking read model: %w", err)
	}
	return nil
}

func (r PostgresRepository) UpdateByBookingID(ctx context.Context, bookingID string, update func(booking *entity.OpsBooking) error) error {
	return updateInTx(ctx, r.db, sql.LevelRepeatableRead, func(ctx context.Context, tx *sqlx.Tx) error {
		booking, err := r.getByID(ctx, tx, bookingID)
		if err != nil {
			return err
		}

		if err = update(booking); err != nil {
			return err
		}

		payload, err := json.Marshal(booking)
		if err != nil {
			return err
		}

		_, err = tx.NamedExecContext(ctx, `
			UPDATE read_model_ops_bookings 
			SET payload = :payload
			WHERE booking_id = :booking_id
			`, map[string]interface{}{
			"booking_id": bookingID,
			"payload":    payload,
		})
		if err != nil {
			return fmt.Errorf("could not update booking read model: %w", err)
		}

		return nil
	})
}

func (r PostgresRepository) UpdateByTicketID(ctx context.Context, ticketID string, update func(booking *entity.OpsBooking) error) error {
	return updateInTx(ctx, r.db, sql.LevelRepeatableRead, func(ctx context.Context, tx *sqlx.Tx) error {
		booking, err := r.GetByTicketID(ctx, tx, ticketID)
		if err != nil {
			return err
		}

		if err = update(booking); err != nil {
			return err
		}

		payload, err := json.Marshal(booking)
		if err != nil {
			return err
		}

		_, err = tx.NamedExecContext(ctx, `
			UPDATE read_model_ops_bookings 
			SET payload = :payload
			WHERE booking_id = :booking_id
			`, map[string]interface{}{
			"booking_id": booking.BookingID,
			"payload":    payload,
		})
		if err != nil {
			return fmt.Errorf("could not update booking read model: %w", err)
		}

		return nil
	})
}

func (r PostgresRepository) GetByTicketID(ctx context.Context, tx *sqlx.Tx, ticketID string) (*entity.OpsBooking, error) {
	if tx == nil {
		return nil, errors.New("tx is nil")
	}
	var payload []byte
	err := tx.GetContext(ctx, &payload, `
		SELECT payload
		FROM read_model_ops_bookings
		WHERE payload->'tickets' ? $1
		`,
		ticketID,
	)
	if err != nil {
		return nil, fmt.Errorf("could not get booking read model: %w", err)
	}

	var booking entity.OpsBooking
	if err = json.Unmarshal(payload, &booking); err != nil {
		return nil, fmt.Errorf("could not unmarshal booking read model: %w", err)
	}

	return &booking, nil
}

func (r PostgresRepository) getByID(ctx context.Context, tx *sqlx.Tx, id string) (*entity.OpsBooking, error) {
	if tx == nil {
		return nil, errors.New("tx is nil")
	}
	var payload []byte
	err := tx.GetContext(ctx, &payload, `
		SELECT payload 
		FROM read_model_ops_bookings 
		WHERE booking_id = $1
		`, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &entity.OpsBooking{}, nil
		}
		return nil, fmt.Errorf("could not get booking read model: %w", err)
	}

	var booking entity.OpsBooking
	if err = json.Unmarshal(payload, &booking); err != nil {
		return nil, fmt.Errorf("could not unmarshal booking read model: %w", err)
	}

	return &booking, nil
}

func updateInTx(
	ctx context.Context,
	db *sqlx.DB,
	isolation sql.IsolationLevel,
	fn func(ctx context.Context, tx *sqlx.Tx) error,
) (err error) {
	tx, err := db.BeginTxx(ctx, &sql.TxOptions{Isolation: isolation})
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = errors.Join(err, rollbackErr)
			}
			return
		}

		err = tx.Commit()
	}()

	return fn(ctx, tx)
}
