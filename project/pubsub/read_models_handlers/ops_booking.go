package read_models_handlers

import (
	"context"
	"errors"
	"time"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"

	"tickets/entity"
)

type Repository interface {
	Store(ctx context.Context, booking entity.OpsBooking) error
	Update(ctx context.Context, bookingID string, update func(booking *entity.OpsBooking) error) error
}

type OpsBookingReadModel struct {
	repo Repository
}

func NewOpsBookingReadModel(repo Repository) OpsBookingReadModel {
	return OpsBookingReadModel{repo: repo}
}

func (r OpsBookingReadModel) OnBookingMade(ctx context.Context, bookingMade *entity.BookingMade) error {
	log.FromContext(ctx).Infof("OpsBookingReadModel: OnBookingMade: %s", bookingMade.BookingID)
	readModel := entity.OpsBooking{
		BookingID:  bookingMade.BookingID,
		BookedAt:   bookingMade.Header.PublishedAt,
		Tickets:    nil,
		LastUpdate: time.Now(),
	}

	return r.repo.Store(ctx, readModel)
}

func (r OpsBookingReadModel) OnTicketReceiptIssued(ctx context.Context, ticketBookingConfirmed *entity.TicketReceiptIssued) error {
	log.FromContext(ctx).Infof("OpsBookingReadModel: OnTicketReceiptIssued: %s", ticketBookingConfirmed.TicketID)
	return r.repo.Update(ctx, ticketBookingConfirmed.TicketID, func(booking *entity.OpsBooking) error {
		ticket, ok := booking.Tickets[ticketBookingConfirmed.TicketID]
		if !ok {
			return errors.New("ticket not found")
		}

		ticket.ReceiptIssuedAt = ticketBookingConfirmed.Header.PublishedAt
		ticket.ReceiptNumber = ticketBookingConfirmed.ReceiptNumber

		booking.Tickets[ticketBookingConfirmed.TicketID] = ticket
		booking.LastUpdate = time.Now()

		return nil
	})
}

func (r OpsBookingReadModel) OnTicketBookingConfirmed(ctx context.Context, ticketBookingConfirmed *entity.TicketBookingConfirmed) error {
	log.FromContext(ctx).Infof("OpsBookingReadModel: OnTicketBookingConfirmed: %s", ticketBookingConfirmed.TicketID)
	return r.repo.Update(ctx, ticketBookingConfirmed.TicketID, func(booking *entity.OpsBooking) error {
		ticket, ok := booking.Tickets[ticketBookingConfirmed.TicketID]
		if !ok {
			return errors.New("ticket not found")
		}

		ticket.PriceAmount = ticketBookingConfirmed.Price.Amount
		ticket.PriceCurrency = ticketBookingConfirmed.Price.Currency
		ticket.CustomerEmail = ticketBookingConfirmed.CustomerEmail
		if ticket.Status != "refunded" {
			ticket.Status = "confirmed"
		}

		booking.Tickets[ticketBookingConfirmed.TicketID] = ticket
		booking.LastUpdate = time.Now()

		return nil
	})
}

func (r OpsBookingReadModel) OnTicketPrinted(ctx context.Context, ticketPrinted *entity.TicketPrinted) error {
	log.FromContext(ctx).Infof("OpsBookingReadModel: OnTicketPrinted: %s", ticketPrinted.TicketID)
	return r.repo.Update(ctx, ticketPrinted.TicketID, func(booking *entity.OpsBooking) error {
		ticket, ok := booking.Tickets[ticketPrinted.TicketID]
		if !ok {
			return errors.New("ticket not found")
		}

		ticket.PrintedAt = ticketPrinted.Header.PublishedAt
		ticket.PrintedFileName = ticketPrinted.FileName

		booking.Tickets[ticketPrinted.TicketID] = ticket
		booking.LastUpdate = time.Now()

		return nil
	})
}

func (r OpsBookingReadModel) OnTicketRefunded(ctx context.Context, ticketRefunded *entity.TicketRefunded) error {
	log.FromContext(ctx).Infof("OpsBookingReadModel: OnTicketRefunded: %s", ticketRefunded.TicketID)
	return r.repo.Update(ctx, ticketRefunded.TicketID, func(booking *entity.OpsBooking) error {
		ticket, ok := booking.Tickets[ticketRefunded.TicketID]
		if !ok {
			return errors.New("ticket not found")
		}

		ticket.Status = "refunded"

		booking.Tickets[ticketRefunded.TicketID] = ticket
		booking.LastUpdate = time.Now()

		return nil
	})
}
