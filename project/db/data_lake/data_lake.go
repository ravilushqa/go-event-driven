package db

import (
	"context"
	"errors"
	"fmt"

	"tickets/entity"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type DataLake struct {
	db *sqlx.DB
}

func NewDataLake(db *sqlx.DB) DataLake {
	if db == nil {
		panic("db is nil")
	}

	return DataLake{db: db}
}

func (s DataLake) StoreEvent(
	ctx context.Context,
	dataLakeEvent entity.DataLakeEvent,
) error {
	_, err := s.db.NamedExecContext(
		ctx,
		`
			INSERT INTO 
			    events (event_id, published_at, event_name, event_payload) 
			VALUES 
			    (:event_id, :published_at, :event_name, :event_payload)`,
		dataLakeEvent,
	)
	var postgresError *pq.Error
	if errors.As(err, &postgresError) && postgresError.Code.Name() == "unique_violation" {
		// handling re-delivery
		return nil
	}
	if err != nil {
		return fmt.Errorf("could not store %s event in data lake: %w", dataLakeEvent.ID, err)
	}

	return nil
}

func (s DataLake) GetEvents(ctx context.Context) ([]entity.DataLakeEvent, error) {
	var events []entity.DataLakeEvent
	err := s.db.SelectContext(ctx, &events, "SELECT * FROM events ORDER BY published_at ASC")
	if err != nil {
		return nil, fmt.Errorf("could not get events from data lake: %w", err)
	}

	return events, nil
}
