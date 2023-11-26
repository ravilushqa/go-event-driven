package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"tickets/entity"
	"tickets/pubsub/bus"
	"tickets/pubsub/outbox"
)

const (
	postgresUniqueValueViolationErrorCode = "23505"
)

type VipBundlePostgresRepository struct {
	db *sqlx.DB
}

func NewVipBundlePostgresRepository(db *sqlx.DB) *VipBundlePostgresRepository {
	if db == nil {
		panic("db must be set")
	}

	return &VipBundlePostgresRepository{db: db}
}

type Executor interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// Add adds vip bundle to the repository. It returns error if vip bundle with given ID already exists.
// It also publishes VipBundleInitialized_v1 event.
func (r VipBundlePostgresRepository) Add(ctx context.Context, vipBundle entity.VipBundle) error {
	payload, err := json.Marshal(vipBundle)
	if err != nil {
		return fmt.Errorf("could not marshal vip bundle: %w", err)
	}

	return UpdateInTx(
		ctx,
		r.db,
		sql.LevelRepeatableRead,
		func(ctx context.Context, tx *sqlx.Tx) error {
			_, err = r.db.ExecContext(ctx, `
				INSERT INTO vip_bundles (vip_bundle_id, booking_id, payload)
				VALUES ($1, $2, $3)
			`, vipBundle.VipBundleID, vipBundle.BookingID, payload)

			if err != nil {
				if isErrorUniqueViolation(err) {
					return nil // De-duplicating
				}
				return fmt.Errorf("could not insert vip bundle: %w", err)
			}

			outboxPublisher, err := outbox.NewPublisherForDb(ctx, tx)
			if err != nil {
				return fmt.Errorf("could not create outbox published: %w", err)
			}

			eventBus, err := bus.NewEventBus(outboxPublisher)
			if err != nil {
				return fmt.Errorf("could not create event bus")
			}

			err = eventBus.Publish(ctx, entity.VipBundleInitialized_v1{
				Header:      entity.NewEventHeader(),
				VipBundleID: vipBundle.VipBundleID,
			})
			if err != nil {
				return fmt.Errorf("could not publish event: %w", err)
			}

			return nil
		},
	)
}

func (r VipBundlePostgresRepository) Get(ctx context.Context, vipBundleID string) (entity.VipBundle, error) {
	return r.vipBundleByID(ctx, vipBundleID, r.db)
}

func (r VipBundlePostgresRepository) vipBundleByID(ctx context.Context, vipBundleID string, db Executor) (entity.VipBundle, error) {
	var payload []byte
	err := db.QueryRowContext(ctx, `
		SELECT payload FROM vip_bundles WHERE vip_bundle_id = $1
	`, vipBundleID).Scan(&payload)
	if err != nil {
		return entity.VipBundle{}, fmt.Errorf("could not get vip bundle by id: %w", err)
	}

	var vipBundle entity.VipBundle
	err = json.Unmarshal(payload, &vipBundle)
	if err != nil {
		return entity.VipBundle{}, fmt.Errorf("could not unmarshal vip bundle: %w", err)
	}

	return vipBundle, nil
}

func (r VipBundlePostgresRepository) GetByBookingID(ctx context.Context, bookingID string) (entity.VipBundle, error) {
	return r.getByBookingID(ctx, bookingID, r.db)
}

func (r VipBundlePostgresRepository) getByBookingID(ctx context.Context, bookingID string, db Executor) (entity.VipBundle, error) {
	var payload []byte
	err := db.QueryRowContext(ctx, `
		SELECT payload FROM vip_bundles WHERE booking_id = $1
	`, bookingID).Scan(&payload)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return entity.VipBundle{}, entity.ErrNotFound
		}
		return entity.VipBundle{}, fmt.Errorf("could not get vip bundle: %w", err)
	}

	var vipBundle entity.VipBundle
	err = json.Unmarshal(payload, &vipBundle)
	if err != nil {
		return entity.VipBundle{}, fmt.Errorf("could not unmarshal vip bundle: %w", err)
	}

	return vipBundle, nil
}

func (r VipBundlePostgresRepository) UpdateByID(ctx context.Context, bookingID string, updateFn func(vipBundle entity.VipBundle) (entity.VipBundle, error)) (entity.VipBundle, error) {
	var vb entity.VipBundle

	err := UpdateInTx(ctx, r.db, sql.LevelSerializable, func(ctx context.Context, tx *sqlx.Tx) error {
		var err error
		vb, err = r.vipBundleByID(ctx, bookingID, tx)
		if err != nil {
			return err
		}

		vb, err = updateFn(vb)
		if err != nil {
			return err
		}

		payload, err := json.Marshal(vb)
		if err != nil {
			return fmt.Errorf("could not marshal vip bundle: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			UPDATE vip_bundles SET payload = $1 WHERE vip_bundle_id = $2
		`, payload, vb.VipBundleID)

		if err != nil {
			return fmt.Errorf("could not update vip bundle: %w", err)
		}

		return nil
	})
	if err != nil {
		return entity.VipBundle{}, fmt.Errorf("could not update vip bundle: %w", err)
	}

	return vb, nil
}

func (r VipBundlePostgresRepository) UpdateByBookingID(ctx context.Context, bookingID string, updateFn func(vipBundle entity.VipBundle) (entity.VipBundle, error)) (entity.VipBundle, error) {
	var vb entity.VipBundle

	err := UpdateInTx(ctx, r.db, sql.LevelSerializable, func(ctx context.Context, tx *sqlx.Tx) error {
		var err error
		vb, err = r.getByBookingID(ctx, bookingID, tx)
		if err != nil {
			return err
		}

		vb, err = updateFn(vb)
		if err != nil {
			return err
		}

		payload, err := json.Marshal(vb)
		if err != nil {
			return fmt.Errorf("could not marshal vip bundle: %w", err)
		}

		_, err = tx.ExecContext(ctx, `
			UPDATE vip_bundles SET payload = $1 WHERE booking_id = $2
		`, payload, vb.BookingID)

		if err != nil {
			return fmt.Errorf("could not update vip bundle: %w", err)
		}

		return nil
	})
	if err != nil {
		return entity.VipBundle{}, fmt.Errorf("could not update vip bundle: %w", err)
	}

	return vb, nil
}

func isErrorUniqueViolation(err error) bool {
	var psqlErr *pq.Error
	return errors.As(err, &psqlErr) && psqlErr.Code == postgresUniqueValueViolationErrorCode
}
