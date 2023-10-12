package tickets

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

func (r *PostgresRepository) Store(ctx context.Context, ticket entity.Ticket) error {
	_, err := r.db.NamedExecContext(ctx, `
		INSERT INTO tickets (ticket_id, price_amount, price_currency, customer_email)
		VALUES (:ticket_id, :price_amount, :price_currency, :customer_email)
		ON CONFLICT DO NOTHING -- ignore if already exists
	`, ticket)
	return err
}

func (r *PostgresRepository) Delete(ctx context.Context, ticketID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM tickets
		WHERE ticket_id = $1
	`, ticketID)
	return err
}

func (r *PostgresRepository) FindAll(ctx context.Context) ([]entity.Ticket, error) {
	var tickets []entity.Ticket
	err := r.db.SelectContext(ctx, &tickets, `
		SELECT ticket_id, price_amount, price_currency, customer_email
		FROM tickets
	`)
	return tickets, err
}