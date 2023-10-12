package pkg

import (
	sql2 "database/sql"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/components/forwarder"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jmoiron/sqlx"
)

const outboxTopic = "events_to_forward"

func NewPsqlSubscriber(db *sqlx.DB, logger watermill.LoggerAdapter) (message.Subscriber, error) {
	var sub message.Subscriber
	sub, err := sql.NewSubscriber(db, sql.SubscriberConfig{
		SchemaAdapter:    sql.DefaultPostgreSQLSchema{},
		OffsetsAdapter:   sql.DefaultPostgreSQLOffsetsAdapter{},
		InitializeSchema: true,
	}, logger)

	return sub, err

}

func NewPsqlPublisher(
	tx *sql2.Tx,
	logger watermill.LoggerAdapter,
) (message.Publisher, error) {
	var publisher message.Publisher
	sqlPublisher, err := sql.NewPublisher(
		tx,
		sql.PublisherConfig{
			SchemaAdapter: sql.DefaultPostgreSQLSchema{},
		},
		logger,
	)
	if err != nil {
		return nil, err
	}

	publisher = forwarder.NewPublisher(sqlPublisher, forwarder.PublisherConfig{
		ForwarderTopic: outboxTopic,
	})

	return publisher, nil
}
