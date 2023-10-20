package events

import (
	"context"

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

func (r PostgresRepository) Store(ctx context.Context, event entity.Event) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO events (event_id, published_at, event_name, event_payload)
		VALUES (:event_id, :published_at, :event_name, :event_payload)
		ON CONFLICT DO NOTHING -- ignore if already exists
	`, event)
	return err
}
