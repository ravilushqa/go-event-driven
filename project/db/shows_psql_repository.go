package db

import (
	"context"

	"github.com/jmoiron/sqlx"

	"tickets/entity"
)

type ShowsPostgresRepository struct {
	db *sqlx.DB
}

func NewShowsPostgresRepository(db *sqlx.DB) *ShowsPostgresRepository {
	return &ShowsPostgresRepository{db: db}
}

func (r *ShowsPostgresRepository) Store(ctx context.Context, show entity.Show) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO shows (show_id, dead_nation_id, number_of_tickets, start_time, title, venue)
		VALUES (:show_id, :dead_nation_id, :number_of_tickets, :start_time, :title, :venue)
		ON CONFLICT DO NOTHING -- ignore if already exists
	`, show)
	return err
}

func (r *ShowsPostgresRepository) Get(ctx context.Context, showID string) (entity.Show, error) {
	var show entity.Show
	err := r.db.GetContext(ctx, &show, `
		SELECT show_id, dead_nation_id, number_of_tickets, start_time, title, venue
		FROM shows
		WHERE show_id = $1
	`, showID)

	return show, err
}
