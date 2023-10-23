package read_model_ops_bookings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/jmoiron/sqlx"
)

type OpsBookingReadModel struct {
	db       *sqlx.DB
	eventBus *cqrs.EventBus
}

func NewOpsBookingReadModel(db *sqlx.DB, eventBus *cqrs.EventBus) OpsBookingReadModel {
	if db == nil {
		panic("db is nil")
	}
	if eventBus == nil {
		panic("eventBus is nil")
	}

	return OpsBookingReadModel{db: db, eventBus: eventBus}
}

func (r OpsBookingReadModel) AllReservations(receiptIssueDateFilter string) ([]entity.OpsBooking, error) {
	query := "SELECT payload FROM read_model_ops_bookings"
	var quaryArgs []any

	if receiptIssueDateFilter != "" {
		// please keep in mind that this is not the most efficient way to do it
		query += fmt.Sprintf(`
			WHERE booking_id IN (
				SELECT booking_id FROM (
					SELECT booking_id, 
						DATE(jsonb_path_query(payload, '$.tickets.*.receipt_issued_at')::text) as receipt_issued_at 
					FROM 
						read_model_ops_bookings
				) bookings_within_date 
				WHERE receipt_issued_at = $1
			)
		`)
		quaryArgs = append(quaryArgs, receiptIssueDateFilter)
	}

	rows, err := r.db.Query(query, quaryArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []entity.OpsBooking
	for rows.Next() {
		var payload []byte
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}

		var reservation entity.OpsBooking
		if err := json.Unmarshal(payload, &reservation); err != nil {
			return nil, err
		}

		result = append(result, reservation)
	}

	return result, nil
}

func (r OpsBookingReadModel) ReservationReadModel(ctx context.Context, bookingID string) (entity.OpsBooking, error) {
	return r.findReadModelByBookingID(ctx, bookingID, r.db)
}

func (r OpsBookingReadModel) OnBookingMade(ctx context.Context, bookingMade *entity.BookingMade_v1) error {
	// this is the first event that should arrive, so we create the read model
	err := r.createReadModel(ctx, entity.OpsBooking{
		BookingID:  bookingMade.BookingID,
		Tickets:    nil,
		LastUpdate: time.Now(),
		BookedAt:   bookingMade.Header.PublishedAt,
	})
	if err != nil {
		return fmt.Errorf("could not create read model: %w", err)
	}

	return nil
}

func (r OpsBookingReadModel) OnTicketBookingConfirmed(ctx context.Context, event *entity.TicketBookingConfirmed_v1) error {
	return r.updateBookingReadModel(
		ctx,
		event.BookingID,
		func(rm entity.OpsBooking) (entity.OpsBooking, error) {

			ticket, ok := rm.Tickets[event.TicketID]
			if !ok {
				// we are using zero-value of OpsTicket
				log.
					FromContext(ctx).
					WithField("ticket_id", event.TicketID).
					Debug("Creating ticket read model for ticket %s")
			}

			ticket.PriceAmount = event.Price.Amount
			ticket.PriceCurrency = event.Price.Currency
			ticket.CustomerEmail = event.CustomerEmail
			ticket.ConfirmedAt = event.Header.PublishedAt

			rm.Tickets[event.TicketID] = ticket

			return rm, nil
		},
	)
}

func (r OpsBookingReadModel) OnTicketRefunded(ctx context.Context, event *entity.TicketRefunded_v1) error {
	return r.updateTicketInBookingReadModel(
		ctx,
		event.TicketID,
		func(rm entity.OpsTicket) (entity.OpsTicket, error) {
			rm.RefundedAt = event.Header.PublishedAt

			return rm, nil
		},
	)
}

func (r OpsBookingReadModel) OnTicketPrinted(ctx context.Context, event *entity.TicketPrinted_v1) error {
	return r.updateTicketInBookingReadModel(
		ctx,
		event.TicketID,
		func(rm entity.OpsTicket) (entity.OpsTicket, error) {
			rm.PrintedAt = event.Header.PublishedAt
			rm.PrintedFileName = event.FileName

			return rm, nil
		},
	)
}

func (r OpsBookingReadModel) OnTicketReceiptIssued(ctx context.Context, issued *entity.TicketReceiptIssued_v1) error {
	return r.updateTicketInBookingReadModel(
		ctx,
		issued.TicketID,
		func(rm entity.OpsTicket) (entity.OpsTicket, error) {
			rm.ReceiptIssuedAt = issued.IssuedAt
			rm.ReceiptNumber = issued.ReceiptNumber

			return rm, nil
		},
	)
}

func (r OpsBookingReadModel) createReadModel(
	ctx context.Context,
	booking entity.OpsBooking,
) (err error) {
	payload, err := json.Marshal(booking)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO 
		    read_model_ops_bookings (payload, booking_id)
		VALUES
			($1, $2)
		ON CONFLICT (booking_id) DO NOTHING; -- read model may be already updated by another event - we don't want to override
`, payload, booking.BookingID)

	if err != nil {
		return fmt.Errorf("could not create read model: %w", err)
	}

	return nil
}

func (r OpsBookingReadModel) updateBookingReadModel(
	ctx context.Context,
	bookingID string,
	updateFunc func(ticket entity.OpsBooking) (entity.OpsBooking, error),
) (err error) {
	return updateInTx(
		ctx,
		r.db,
		sql.LevelRepeatableRead,
		func(ctx context.Context, tx *sqlx.Tx) error {
			rm, err := r.findReadModelByBookingID(ctx, bookingID, tx)
			if errors.Is(err, sql.ErrNoRows) {
				// events arrived out of order - it should spin until the read model is created
				return fmt.Errorf("read model for booking %s not exist yet", bookingID)
			} else if err != nil {
				return fmt.Errorf("could not find read model: %w", err)
			}

			updatedRm, err := updateFunc(rm)
			if err != nil {
				return err
			}

			err = r.updateReadModel(ctx, tx, updatedRm)
			if err != nil {
				return err
			}

			err = r.eventBus.Publish(ctx, entity.InternalOpsReadModelUpdated{
				Header:    entity.NewEventHeader(),
				BookingID: bookingID,
			})
			if err != nil {
				log.FromContext(ctx).Errorf("could not publish event InternalOpsReadModelUpdated: %s", err)
			}
			return nil
		},
	)
}

func (r OpsBookingReadModel) updateTicketInBookingReadModel(
	ctx context.Context,
	ticketID string,
	updateFunc func(ticket entity.OpsTicket) (entity.OpsTicket, error),
) (err error) {
	return updateInTx(
		ctx,
		r.db,
		sql.LevelRepeatableRead,
		func(ctx context.Context, tx *sqlx.Tx) error {
			rm, err := r.findReadModelByTicketID(ctx, ticketID, tx)
			if errors.Is(err, sql.ErrNoRows) {
				// events arrived out of order - it should spin until the read model is created
				return fmt.Errorf("read model for ticket %s not exist yet", ticketID)
			} else if err != nil {
				return fmt.Errorf("could not find read model: %w", err)
			}

			ticket, _ := rm.Tickets[ticketID]

			updatedRm, err := updateFunc(ticket)
			if err != nil {
				return err
			}

			rm.Tickets[ticketID] = updatedRm

			return r.updateReadModel(ctx, tx, rm)
		},
	)
}

func (r OpsBookingReadModel) updateReadModel(
	ctx context.Context,
	tx *sqlx.Tx,
	rm entity.OpsBooking,
) error {
	rm.LastUpdate = time.Now()

	payload, err := json.Marshal(rm)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO 
			read_model_ops_bookings (payload, booking_id)
		VALUES
			($1, $2)
		ON CONFLICT (booking_id) DO UPDATE SET payload = excluded.payload;
		`, payload, rm.BookingID)
	if err != nil {
		return fmt.Errorf("could not update read model: %w", err)
	}

	return nil
}

func (r OpsBookingReadModel) findReadModelByTicketID(
	ctx context.Context,
	ticketID string,
	db dbExecutor,
) (entity.OpsBooking, error) {
	var payload []byte

	err := db.QueryRowContext(
		ctx,
		"SELECT payload FROM read_model_ops_bookings WHERE payload::jsonb -> 'tickets' ? $1",
		ticketID,
	).Scan(&payload)
	if err != nil {
		return entity.OpsBooking{}, err
	}

	return r.unmarshalReadModelFromDB(payload)
}

func (r OpsBookingReadModel) findReadModelByBookingID(
	ctx context.Context,
	bookingID string,
	db dbExecutor,
) (entity.OpsBooking, error) {
	var payload []byte

	err := db.QueryRowContext(
		ctx,
		"SELECT payload FROM read_model_ops_bookings WHERE booking_id = $1",
		bookingID,
	).Scan(&payload)
	if err != nil {
		return entity.OpsBooking{}, err
	}

	return r.unmarshalReadModelFromDB(payload)
}

func (r OpsBookingReadModel) unmarshalReadModelFromDB(payload []byte) (entity.OpsBooking, error) {
	var dbReadModel entity.OpsBooking
	if err := json.Unmarshal(payload, &dbReadModel); err != nil {
		return entity.OpsBooking{}, err
	}

	if dbReadModel.Tickets == nil {
		dbReadModel.Tickets = map[string]entity.OpsTicket{}
	}

	return dbReadModel, nil
}

type dbExecutor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
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
