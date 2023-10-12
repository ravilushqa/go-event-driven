package outbox

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	watermillSQL "github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
)

func NewPostgresSubscriber(db *sql.DB, logger watermill.LoggerAdapter) *watermillSQL.Subscriber {
	sub, err := watermillSQL.NewSubscriber(
		db,
		watermillSQL.SubscriberConfig{
			PollInterval:     time.Millisecond * 100,
			InitializeSchema: true,
			SchemaAdapter:    watermillSQL.DefaultPostgreSQLSchema{},
			OffsetsAdapter:   watermillSQL.DefaultPostgreSQLOffsetsAdapter{},
		},
		logger,
	)
	if err != nil {
		panic(fmt.Errorf("failed to create new watermill sql subscriber: %w", err))
	}

	return sub
}
