package event

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"

	"tickets/db/read_model_ops_bookings"
	"tickets/entity"
)

type OpsBookingHandlers struct {
	repo read_model_ops_bookings.OpsBookingReadModel
}

func NewOpsBookingHandlers(repo read_model_ops_bookings.OpsBookingReadModel) OpsBookingHandlers {
	return OpsBookingHandlers{repo: repo}

}

func (r OpsBookingHandlers) OnBookingMade(ctx context.Context, bookingMade *entity.BookingMade_v1) error {
	// this is the first event that should arrive, so we create the read model
	err := r.repo.CreateReadModel(ctx, entity.OpsBooking{
		BookingID:  bookingMade.BookingID,
		Tickets:    nil,
		LastUpdate: time.Now(),
		BookedAt:   bookingMade.Header.PublishedAt,
	})
	if err != nil {
		return fmt.Errorf("could not create read model: %w", err)
	}

	return nil
}

func (r OpsBookingHandlers) OnTicketBookingConfirmed(ctx context.Context, event *entity.TicketBookingConfirmed_v1) error {
	return r.repo.UpdateBookingReadModel(
		ctx,
		event.BookingID,
		func(rm entity.OpsBooking) (entity.OpsBooking, error) {
			ticket, ok := rm.Tickets[event.TicketID]
			if !ok {
				// we are using zero-value of OpsTicket
				log.
					FromContext(ctx).
					WithField("ticket_id", event.TicketID).
					Debug("Creating ticket read model for ticket %s")
			}

			ticket.PriceAmount = event.Price.Amount
			ticket.PriceCurrency = event.Price.Currency
			ticket.CustomerEmail = event.CustomerEmail
			ticket.ConfirmedAt = event.Header.PublishedAt

			rm.Tickets[event.TicketID] = ticket

			return rm, nil
		},
	)
}

func (r OpsBookingHandlers) OnTicketRefunded(ctx context.Context, event *entity.TicketRefunded_v1) error {
	return r.repo.UpdateTicketInBookingReadModel(
		ctx,
		event.TicketID,
		func(rm entity.OpsTicket) (entity.OpsTicket, error) {
			rm.RefundedAt = event.Header.PublishedAt

			return rm, nil
		},
	)
}

func (r OpsBookingHandlers) OnTicketPrinted(ctx context.Context, event *entity.TicketPrinted_v1) error {
	return r.repo.UpdateTicketInBookingReadModel(
		ctx,
		event.TicketID,
		func(rm entity.OpsTicket) (entity.OpsTicket, error) {
			rm.PrintedAt = event.Header.PublishedAt
			rm.PrintedFileName = event.FileName

			return rm, nil
		},
	)
}

func (r OpsBookingHandlers) OnTicketReceiptIssued(ctx context.Context, issued *entity.TicketReceiptIssued_v1) error {
	return r.repo.UpdateTicketInBookingReadModel(
		ctx,
		issued.TicketID,
		func(rm entity.OpsTicket) (entity.OpsTicket, error) {
			rm.ReceiptIssuedAt = issued.IssuedAt
			rm.ReceiptNumber = issued.ReceiptNumber

			return rm, nil
		},
	)
}
