package http

import (
	"encoding/json"
	"net/http"

	"github.com/ThreeDotsLabs/watermill"
	watermillMessage "github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"tickets/entity"
	"tickets/pkg"
)

func (s Server) PostBookTickets(c echo.Context) error {
	var request postBookTicketsRequest
	err := c.Bind(&request)
	if err != nil {
		return err
	}

	booking := entity.Booking{
		BookingID:       uuid.NewString(),
		ShowID:          request.ShowID,
		NumberOfTickets: request.NumberOfTickets,
		CustomerEmail:   request.CustomerEmail,
	}

	tx, err := s.db.BeginTx(c.Request().Context(), nil)
	if err != nil {
		return err
	}
	err = s.bookingsRepo.Store(c.Request().Context(), booking)
	if err != nil {
		return err
	}

	publisher, err := pkg.NewPsqlPublisher(tx, watermill.NopLogger{})
	if err != nil {
		return err
	}

	event := entity.BookingMade{
		Header:          entity.NewEventHeader(),
		BookingID:       booking.BookingID,
		NumberOfTickets: booking.NumberOfTickets,
		CustomerEmail:   booking.CustomerEmail,
		ShowID:          booking.ShowID,
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msgToPublish := watermillMessage.NewMessage(uuid.NewString(), eventBytes)
	err = publisher.Publish("BookingMade", msgToPublish)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}
	return c.JSON(http.StatusCreated, postBookTicketsResponse{
		BookingID: booking.BookingID,
	})
}
