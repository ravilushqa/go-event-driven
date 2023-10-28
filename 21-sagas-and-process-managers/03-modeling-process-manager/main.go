package main

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type EventHeader struct {
	ID             string    `json:"id"`
	PublishedAt    time.Time `json:"published_at"`
	IdempotencyKey string    `json:"idempotency_key"`
}

func NewEventHeader() EventHeader {
	return EventHeader{
		ID:             uuid.NewString(),
		PublishedAt:    time.Now(),
		IdempotencyKey: uuid.NewString(),
	}
}

type Money struct {
	Amount   string `json:"amount" db:"amount"`
	Currency string `json:"currency" db:"currency"`
}

type BookShowTickets struct {
	BookingID uuid.UUID `json:"booking_id"`

	CustomerEmail   string    `json:"customer_email"`
	NumberOfTickets int       `json:"number_of_tickets"`
	ShowId          uuid.UUID `json:"show_id"`
}

type BookFlight struct {
	CustomerEmail  string    `json:"customer_email"`
	FlightID       uuid.UUID `json:"to_flight_id"`
	Passengers     []string  `json:"passengers"`
	ReferenceID    string    `json:"reference_id"`
	IdempotencyKey string    `json:"idempotency_key"`
}

type BookTaxi struct {
	CustomerEmail      string `json:"customer_email"`
	CustomerName       string `json:"customer_name"`
	NumberOfPassengers int    `json:"number_of_passengers"`
	ReferenceID        string `json:"reference_id"`
	IdempotencyKey     string `json:"idempotency_key"`
}

type CancelFlightTickets struct {
	FlightTicketIDs []uuid.UUID `json:"flight_ticket_id"`
}

type RefundTicket struct {
	Header EventHeader `json:"header"`

	TicketID string `json:"ticket_id"`
}

type BookingMade_v1 struct {
	Header EventHeader `json:"header"`

	NumberOfTickets int `json:"number_of_tickets"`

	BookingID uuid.UUID `json:"booking_id"`

	CustomerEmail string    `json:"customer_email"`
	ShowId        uuid.UUID `json:"show_id"`
}

type TicketBookingConfirmed_v1 struct {
	Header EventHeader `json:"header"`

	TicketID      string `json:"ticket_id"`
	CustomerEmail string `json:"customer_email"`
	Price         Money  `json:"price"`

	BookingID string `json:"booking_id"`
}

type VipBundleInitialized_v1 struct {
	Header EventHeader `json:"header"`

	VipBundleID uuid.UUID `json:"vip_bundle_id"`
}

type BookingFailed_v1 struct {
	Header EventHeader `json:"header"`

	BookingID     uuid.UUID `json:"booking_id"`
	FailureReason string    `json:"failure_reason"`
}

type FlightBooked_v1 struct {
	Header EventHeader `json:"header"`

	FlightID  uuid.UUID   `json:"flight_id"`
	TicketIDs []uuid.UUID `json:"flight_tickets_ids"`

	ReferenceID string `json:"reference_id"`
}

type FlightBookingFailed_v1 struct {
	Header EventHeader `json:"header"`

	FlightID      uuid.UUID `json:"flight_id"`
	FailureReason string    `json:"failure_reason"`

	ReferenceID string `json:"reference_id"`
}

type TaxiBooked_v1 struct {
	Header EventHeader `json:"header"`

	TaxiBookingID uuid.UUID `json:"taxi_booking_id"`

	ReferenceID string `json:"reference_id"`
}

type VipBundleFinalized_v1 struct {
	Header EventHeader `json:"header"`

	VipBundleID uuid.UUID `json:"vip_bundle_id"`
}

type TaxiBookingFailed_v1 struct {
	Header EventHeader `json:"header"`

	FailureReason string `json:"failure_reason"`

	ReferenceID string `json:"reference_id"`
}

type CommandBus interface {
	Send(ctx context.Context, command any) error
}

type EventBus interface {
	Publish(ctx context.Context, event any) error
}

type VipBundle struct {
	VipBundleID uuid.UUID `json:"vip_bundle_id"`

	BookingID       uuid.UUID  `json:"booking_id"`
	CustomerEmail   string     `json:"customer_email"`
	NumberOfTickets int        `json:"number_of_tickets"`
	ShowId          uuid.UUID  `json:"show_id"`
	BookingMadeAt   *time.Time `json:"booking_made_at"`

	TicketIDs []uuid.UUID `json:"ticket_ids"`

	Passengers []string `json:"passengers"`

	InboundFlightID         uuid.UUID   `json:"inbound_flight_id"`
	InboundFlightBookedAt   *time.Time  `json:"inbound_flight_booked_at"`
	InboundFlightTicketsIDs []uuid.UUID `json:"inbound_flight_tickets_ids"`

	ReturnFlightID         uuid.UUID   `json:"return_flight_id"`
	ReturnFlightBookedAt   *time.Time  `json:"return_flight_booked_at"`
	ReturnFlightTicketsIDs []uuid.UUID `json:"return_flight_tickets_ids"`

	TaxiBookedAt  *time.Time `json:"taxi_booked_at"`
	TaxiBookingID *uuid.UUID `json:"taxi_booking_id"`

	IsFinalized bool `json:"finalized"`
	Failed      bool `json:"failed"`
}

func NewVipBundle(
	vipBundleID uuid.UUID,
	bookingID uuid.UUID,
	customerEmail string,
	numberOfTickets int,
	showId uuid.UUID,
	passengers []string,
	inboundFlightID uuid.UUID,
	returnFlightID uuid.UUID,
) (*VipBundle, error) {
	if vipBundleID == uuid.Nil {
		return nil, fmt.Errorf("vip bundle id must be set")
	}
	if bookingID == uuid.Nil {
		return nil, fmt.Errorf("booking id must be set")
	}
	if customerEmail == "" {
		return nil, fmt.Errorf("customer email must be set")
	}
	if numberOfTickets <= 0 {
		return nil, fmt.Errorf("number of tickets must be greater than 0")
	}
	if showId == uuid.Nil {
		return nil, fmt.Errorf("show id must be set")
	}
	if numberOfTickets != len(passengers) {
		return nil, fmt.Errorf("number of tickets and passengers count mismatch")
	}
	if inboundFlightID == uuid.Nil {
		return nil, fmt.Errorf("inbound flight id must be set")
	}
	if returnFlightID == uuid.Nil {
		return nil, fmt.Errorf("return flight id must be set")
	}

	return &VipBundle{
		VipBundleID:     vipBundleID,
		BookingID:       bookingID,
		CustomerEmail:   customerEmail,
		NumberOfTickets: numberOfTickets,
		ShowId:          showId,
		Passengers:      passengers,
		InboundFlightID: inboundFlightID,
		ReturnFlightID:  returnFlightID,
	}, nil
}

type VipBundleRepository interface {
	Add(ctx context.Context, vipBundle VipBundle) error
	Get(ctx context.Context, vipBundleID uuid.UUID) (VipBundle, error)
	GetByBookingID(ctx context.Context, bookingID uuid.UUID) (VipBundle, error)

	UpdateByID(
		ctx context.Context,
		bookingID uuid.UUID,
		updateFn func(vipBundle VipBundle) (VipBundle, error),
	) (VipBundle, error)

	UpdateByBookingID(
		ctx context.Context,
		bookingID uuid.UUID,
		updateFn func(vipBundle VipBundle) (VipBundle, error),
	) (VipBundle, error)
}

type VipBundleProcessManager struct {
	commandBus CommandBus
	eventBus   EventBus
	repository VipBundleRepository
}

func NewVipBundleProcessManager(
	commandBus CommandBus,
	eventBus EventBus,
	repository VipBundleRepository,
) *VipBundleProcessManager {
	return &VipBundleProcessManager{
		commandBus: commandBus,
		eventBus:   eventBus,
		repository: repository,
	}
}

func (v VipBundleProcessManager) OnVipBundleInitialized(ctx context.Context, event *VipBundleInitialized_v1) error {
	vb, err := v.repository.Get(ctx, event.VipBundleID)
	if err != nil {
		return err
	}

	return v.commandBus.Send(ctx, BookShowTickets{
		BookingID:       vb.BookingID,
		CustomerEmail:   vb.CustomerEmail,
		NumberOfTickets: vb.NumberOfTickets,
		ShowId:          vb.ShowId,
	})

}

func (v VipBundleProcessManager) OnBookingMade(ctx context.Context, event *BookingMade_v1) error {
	vb, err := v.repository.UpdateByBookingID(ctx, event.BookingID, func(vipBundle VipBundle) (VipBundle, error) {
		vipBundle.BookingMadeAt = &event.Header.PublishedAt
		return vipBundle, nil
	})
	if err != nil {
		return err
	}

	return v.commandBus.Send(ctx, BookFlight{
		CustomerEmail:  vb.CustomerEmail,
		FlightID:       vb.InboundFlightID,
		Passengers:     vb.Passengers,
		ReferenceID:    vb.VipBundleID.String(),
		IdempotencyKey: event.Header.IdempotencyKey,
	})

}

func (v VipBundleProcessManager) OnTicketBookingConfirmed(ctx context.Context, event *TicketBookingConfirmed_v1) error {
	bookingUUID, err := uuid.Parse(event.BookingID)
	if err != nil {
		return err
	}
	_, err = v.repository.UpdateByBookingID(ctx, bookingUUID, func(vb VipBundle) (VipBundle, error) {
		ticketUUID, err := uuid.Parse(event.TicketID)
		if err != nil {
			return vb, err
		}
		vb.TicketIDs = append(vb.TicketIDs, ticketUUID)

		return vb, nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (v VipBundleProcessManager) OnBookingFailed(ctx context.Context, event *BookingFailed_v1) error {
	_, err := v.repository.UpdateByBookingID(ctx, event.BookingID, func(vb VipBundle) (VipBundle, error) {
		vb.IsFinalized = true
		vb.Failed = true

		return vb, nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (v VipBundleProcessManager) OnFlightBooked(ctx context.Context, event *FlightBooked_v1) error {
	id, err := uuid.Parse(event.ReferenceID)
	if err != nil {
		return err
	}
	vb, err := v.repository.UpdateByID(ctx, id, func(vb VipBundle) (VipBundle, error) {
		switch event.FlightID {
		case vb.InboundFlightID:
			vb.InboundFlightBookedAt = &event.Header.PublishedAt
			vb.InboundFlightTicketsIDs = event.TicketIDs
		case vb.ReturnFlightID:
			vb.ReturnFlightBookedAt = &event.Header.PublishedAt
			vb.ReturnFlightTicketsIDs = event.TicketIDs
		default:
			return vb, fmt.Errorf("unexpected FlightID: %s", event.FlightID)
		}
		return vb, nil
	})
	if err != nil {
		return err
	}

	switch event.FlightID {
	case vb.InboundFlightID:
		return v.commandBus.Send(ctx, BookFlight{
			CustomerEmail:  vb.CustomerEmail,
			FlightID:       vb.ReturnFlightID,
			Passengers:     vb.Passengers,
			ReferenceID:    vb.VipBundleID.String(),
			IdempotencyKey: event.Header.IdempotencyKey,
		})
	case vb.ReturnFlightID:
		return v.commandBus.Send(ctx, BookTaxi{
			CustomerEmail:      vb.CustomerEmail,
			CustomerName:       vb.Passengers[0],
			NumberOfPassengers: len(vb.Passengers),
			ReferenceID:        vb.VipBundleID.String(),
			IdempotencyKey:     event.Header.IdempotencyKey,
		})
	default:
		return fmt.Errorf("unexpected FlightID: %s", event.FlightID)
	}
}

func (v VipBundleProcessManager) OnFlightBookingFailed(ctx context.Context, event *FlightBookingFailed_v1) error {
	id, err := uuid.Parse(event.ReferenceID)
	if err != nil {
		return err
	}

	vb, err := v.repository.Get(ctx, id)
	if err != nil {
		return err
	}

	if len(vb.TicketIDs) != vb.NumberOfTickets {
		return fmt.Errorf("TicketBookingConfirmed_v1 was not handled yet")
	}

	vb, err = v.repository.UpdateByID(ctx, id, func(vb VipBundle) (VipBundle, error) {
		vb.IsFinalized = true
		vb.Failed = true
		return vb, nil
	})
	if err != nil {
		return err
	}

	for _, ticketID := range vb.TicketIDs {
		err := v.commandBus.Send(ctx, RefundTicket{
			Header:   NewEventHeader(),
			TicketID: ticketID.String(),
		})
		if err != nil {
			return err
		}
	}

	if vb.InboundFlightBookedAt != nil {
		err := v.commandBus.Send(ctx, CancelFlightTickets{
			FlightTicketIDs: vb.InboundFlightTicketsIDs,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (v VipBundleProcessManager) OnTaxiBooked(ctx context.Context, event *TaxiBooked_v1) error {
	id, err := uuid.Parse(event.ReferenceID)
	if err != nil {
		return err
	}
	_, err = v.repository.UpdateByID(ctx, id, func(vb VipBundle) (VipBundle, error) {
		vb.TaxiBookingID = &event.TaxiBookingID
		vb.TaxiBookedAt = &event.Header.PublishedAt
		vb.IsFinalized = true
		return vb, nil
	})
	if err != nil {
		return err
	}

	err = v.eventBus.Publish(ctx, VipBundleFinalized_v1{
		Header:      NewEventHeader(),
		VipBundleID: id,
	})
	if err != nil {
		return err
	}

	return nil
}

func (v VipBundleProcessManager) OnTaxiBookingFailed(ctx context.Context, event *TaxiBookingFailed_v1) error {
	id, err := uuid.Parse(event.ReferenceID)
	if err != nil {
		return err
	}

	vb, err := v.repository.UpdateByID(ctx, id, func(vb VipBundle) (VipBundle, error) {
		vb.IsFinalized = true
		vb.Failed = true

		return vb, nil
	})

	for _, ticketID := range vb.TicketIDs {
		err := v.commandBus.Send(ctx, RefundTicket{
			Header:   NewEventHeader(),
			TicketID: ticketID.String(),
		})
		if err != nil {
			return err
		}
	}

	err = v.commandBus.Send(ctx, CancelFlightTickets{
		FlightTicketIDs: vb.InboundFlightTicketsIDs,
	})
	if err != nil {
		return err
	}

	err = v.commandBus.Send(ctx, CancelFlightTickets{
		FlightTicketIDs: vb.ReturnFlightTicketsIDs,
	})
	if err != nil {
		return err
	}

	return nil
}
