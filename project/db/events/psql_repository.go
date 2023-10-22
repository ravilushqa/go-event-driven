package events

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"tickets/entity"
)

type PostgresRepository struct {
	db *sqlx.DB
}

func NewPostgresRepository(db *sqlx.DB) PostgresRepository {
	if db == nil {
		panic("db is nil")
	}

	return PostgresRepository{db: db}
}

func (r PostgresRepository) Store(ctx context.Context,
	eventID string,
	eventHeader entity.EventHeader,
	eventName string,
	payload []byte,
) error {
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO events
			(event_id, published_at, event_name, event_payload)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (event_id) DO NOTHING;
		`,
		eventID,
		eventHeader.PublishedAt,
		eventName,
		payload,
	)
	if err != nil {
		return fmt.Errorf("could not store %s event in data lake: %w", eventID, err)
	}

	return nil
}
