package main

import (
	"encoding/json"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type Header struct {
	ID         string `json:"id"`
	EventName  string `json:"event_name"`
	OccurredAt string `json:"occurred_at"`
}

type ProductOutOfStock struct {
	Header    Header `json:"header"`
	ProductID string `json:"product_id"`
}

type ProductBackInStock struct {
	Header    Header `json:"header"`
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

type Publisher struct {
	pub message.Publisher
}

func NewPublisher(pub message.Publisher) Publisher {
	return Publisher{
		pub: pub,
	}
}

func (p Publisher) PublishProductOutOfStock(productID string) error {
	event := ProductOutOfStock{
		Header:    newHeader("ProductOutOfStock"),
		ProductID: productID,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)

	return p.pub.Publish("product-updates", msg)
}

func (p Publisher) PublishProductBackInStock(productID string, quantity int) error {
	event := ProductBackInStock{
		Header:    newHeader("ProductBackInStock"),
		ProductID: productID,
		Quantity:  quantity,
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := message.NewMessage(watermill.NewUUID(), payload)

	return p.pub.Publish("product-updates", msg)
}

func newHeader(eventName string) Header {
	return Header{
		ID:         watermill.NewUUID(),
		EventName:  eventName,
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
	}
}
