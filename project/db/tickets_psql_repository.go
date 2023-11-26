package db

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"tickets/entity"
)

type TicketsPostgresRepository struct {
	db *sqlx.DB
}

func NewTicketsPostgresRepository(db *sqlx.DB) *TicketsPostgresRepository {
	return &TicketsPostgresRepository{db: db}
}

func (r *TicketsPostgresRepository) Store(ctx context.Context, ticket entity.Ticket) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO tickets (ticket_id, price_amount, price_currency, customer_email)
		VALUES (:ticket_id, :price_amount, :price_currency, :customer_email)
		ON CONFLICT DO NOTHING -- ignore if already exists
	`, ticket)
	return err
}

func (r *TicketsPostgresRepository) Delete(ctx context.Context, ticketID string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE tickets
		SET deleted_at = NOW()
		WHERE ticket_id = $1
	`, ticketID)

	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("ticket with ID %s not found", ticketID)
	}

	return nil
}

func (r *TicketsPostgresRepository) FindAll(ctx context.Context) ([]entity.Ticket, error) {
	var tickets []entity.Ticket
	err := r.db.SelectContext(ctx, &tickets, `
		SELECT ticket_id, price_amount, price_currency, customer_email
		FROM tickets
		WHERE deleted_at IS NULL
	`)
	return tickets, err
}
