package shows

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

func (r *PostgresRepository) Store(ctx context.Context, show entity.Show) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO shows (show_id, dead_nation_id, number_of_tickets, start_time, title, venue)
		VALUES (:show_id, :dead_nation_id, :number_of_tickets, :start_time, :title, :venue)
		ON CONFLICT DO NOTHING -- ignore if already exists
	`, show)
	return err
}
