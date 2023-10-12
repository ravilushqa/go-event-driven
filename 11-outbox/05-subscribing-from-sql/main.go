package main

import (
	"context"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-sql/v2/pkg/sql"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func SubscribeForMessages(db *sqlx.DB, topic string, logger watermill.LoggerAdapter) (<-chan *message.Message, error) {
	subscriber, err := sql.NewSubscriber(db, sql.SubscriberConfig{
		SchemaAdapter:    sql.DefaultPostgreSQLSchema{},
		OffsetsAdapter:   sql.DefaultPostgreSQLOffsetsAdapter{},
		InitializeSchema: true,
	}, logger)
	if err != nil {
		return nil, err
	}

	return subscriber.Subscribe(context.Background(), topic)
}
