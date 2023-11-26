package http

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"tickets/entity"
)

type vipBundleRequest struct {
	CustomerEmail   string   `json:"customer_email"`
	InboundFlightID string   `json:"inbound_flight_id"`
	NumberOfTickets int      `json:"number_of_tickets"`
	Passengers      []string `json:"passengers"`
	ReturnFlightID  string   `json:"return_flight_id"`
	ShowID          string   `json:"show_id"`
}

type vipBundleResponse struct {
	BookingID   string `json:"booking_id"`
	VipBundleID string `json:"vip_bundle_id"`
}

func (s Server) PostBookVipBundle(c echo.Context) error {
	var r vipBundleRequest
	err := c.Bind(&r)
	if err != nil {
		return err
	}

	vb := entity.VipBundle{
		VipBundleID:     uuid.NewString(),
		BookingID:       uuid.NewString(),
		ShowId:          r.ShowID,
		ReturnFlightID:  r.ReturnFlightID,
		CustomerEmail:   r.CustomerEmail,
		NumberOfTickets: r.NumberOfTickets,
		Passengers:      r.Passengers,
		InboundFlightID: r.InboundFlightID,
	}

	err = s.vipBundleRepo.Add(c.Request().Context(), vb)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, vipBundleResponse{
		VipBundleID: vb.VipBundleID,
		BookingID:   vb.BookingID,
	})
}
