package entity

import (
	"time"
)

type DataLakeEvent struct {
	ID          string    `db:"event_id"`
	PublishedAt time.Time `db:"published_at"`
	Name        string    `db:"event_name"`
	Payload     []byte    `db:"event_payload"`
}
