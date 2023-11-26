package migrations

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"tickets/db"
	"tickets/entity"
	"tickets/pubsub/handlers/event"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func MigrateReadModel(ctx context.Context, dl db.DataLake, rm event.OpsBookingHandlers) error {
	var events []entity.DataLakeEvent

	logger := log.FromContext(ctx)
	logger.Info("Migrating read model")

	timeout := time.Now().Add(time.Second * 10)

	// events are not immediately available in the data lake, so we need to wait for them
	for {
		var err error
		events, err = dl.GetEvents(ctx)
		if err != nil {
			return fmt.Errorf("could not get events from data lake: %w", err)
		}
		if len(events) > 0 {
			break
		}

		if time.Now().After(timeout) {
			return fmt.Errorf("timeout while waiting for events in data lake")
		}

		time.Sleep(time.Millisecond * 100)
	}

	logger.WithField("events_count", len(events)).Info("Has events to migrate")

	for _, event := range events {
		start := time.Now()

		logger := log.FromContext(ctx)
		logger.WithFields(logrus.Fields{
			"event_name": event.Name,
			"event_id":   event.ID,
		}).Info("Migrating event")

		err := migrateEvent(ctx, event, rm)
		if err != nil {
			return fmt.Errorf("could not migrate event %s (%s): %w", event.ID, event.Name, err)
		}

		logger.WithField("duration", time.Since(start)).Info("Event migrated")
	}

	return nil
}

// Lucky, the events stored in the Data Lake are the same as the ones from entities package...
// but probably you are not.
//
// To show you the principle, we implemented v0 events in a way that we would do when the format doesn't match.

type bookingMade_v0 struct {
	Header entity.EventHeader `json:"header"`

	NumberOfTickets int `json:"number_of_tickets"`

	BookingID uuid.UUID `json:"booking_id"`

	CustomerEmail string    `json:"customer_email"`
	ShowId        uuid.UUID `json:"show_id"`
}

type ticketBookingConfirmed_v0 struct {
	Header entity.EventHeader `json:"header"`

	TicketID      string       `json:"ticket_id"`
	CustomerEmail string       `json:"customer_email"`
	Price         entity.Money `json:"price"`

	BookingID string `json:"booking_id"`
}

type ticketReceiptIssued_v0 struct {
	Header entity.EventHeader `json:"header"`

	TicketID      string `json:"ticket_id"`
	ReceiptNumber string `json:"receipt_number"`

	IssuedAt time.Time `json:"issued_at"`
}

type ticketPrinted_v0 struct {
	Header entity.EventHeader `json:"header"`

	TicketID string `json:"ticket_id"`
	FileName string `json:"file_name"`
}

type ticketRefunded_v0 struct {
	Header entity.EventHeader `json:"header"`

	TicketID string `json:"ticket_id"`
}

func migrateEvent(ctx context.Context, event entity.DataLakeEvent, rm event.OpsBookingHandlers) error {
	switch event.Name {
	case "BookingMade_v0":
		bookingMade, err := unmarshalDataLakeEvent[bookingMade_v0](event)
		if err != nil {
			return err
		}

		return rm.OnBookingMade(ctx, &entity.BookingMade_v1{
			// you should map v0 event to your v1 event here
			Header:          bookingMade.Header,
			NumberOfTickets: bookingMade.NumberOfTickets,
			BookingID:       bookingMade.BookingID.String(),
			CustomerEmail:   bookingMade.CustomerEmail,
			ShowID:          bookingMade.ShowId.String(),
		})
	case "TicketBookingConfirmed_v0":
		bookingConfirmedEvent, err := unmarshalDataLakeEvent[ticketBookingConfirmed_v0](event)
		if err != nil {
			return err
		}

		return rm.OnTicketBookingConfirmed(ctx, &entity.TicketBookingConfirmed_v1{
			// you should map v0 event to your v1 event here
			Header:        bookingConfirmedEvent.Header,
			TicketID:      bookingConfirmedEvent.TicketID,
			CustomerEmail: bookingConfirmedEvent.CustomerEmail,
			Price:         bookingConfirmedEvent.Price,
			BookingID:     bookingConfirmedEvent.BookingID,
		})
	case "TicketReceiptIssued_v0":
		receiptIssuedEvent, err := unmarshalDataLakeEvent[ticketReceiptIssued_v0](event)
		if err != nil {
			return err
		}

		return rm.OnTicketReceiptIssued(ctx, &entity.TicketReceiptIssued_v1{
			// you should map v0 event to your v1 event here
			Header:        receiptIssuedEvent.Header,
			TicketID:      receiptIssuedEvent.TicketID,
			ReceiptNumber: receiptIssuedEvent.ReceiptNumber,
			IssuedAt:      receiptIssuedEvent.IssuedAt,
		})
	case "TicketPrinted_v0":
		ticketPrintedEvent, err := unmarshalDataLakeEvent[ticketPrinted_v0](event)
		if err != nil {
			return err
		}

		return rm.OnTicketPrinted(ctx, &entity.TicketPrinted_v1{
			// you should map v0 event to your v1 event here
			Header:   ticketPrintedEvent.Header,
			TicketID: ticketPrintedEvent.TicketID,
			FileName: ticketPrintedEvent.FileName,
		})
	case "TicketRefunded_v0":
		ticketRefundedEvent, err := unmarshalDataLakeEvent[ticketRefunded_v0](event)
		if err != nil {
			return err
		}

		return rm.OnTicketRefunded(ctx, &entity.TicketRefunded_v1{
			// you should map v0 event to your v1 event here
			Header:   ticketRefundedEvent.Header,
			TicketID: ticketRefundedEvent.TicketID,
		})
	default:
		return fmt.Errorf("unknown event %s", event.Name)
	}
}

func unmarshalDataLakeEvent[T any](event entity.DataLakeEvent) (*T, error) {
	eventInstance := new(T)

	err := json.Unmarshal(event.Payload, &eventInstance)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal event %s: %w", event.Name, err)
	}

	return eventInstance, nil
}
