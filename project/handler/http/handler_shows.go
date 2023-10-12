package http

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"tickets/entity"
)

type postShowsRequest struct {
	DeadNationID    string    `json:"dead_nation_id"`
	NumberOfTickets int       `json:"number_of_tickets"`
	StartTime       time.Time `json:"start_time"`
	Title           string    `json:"title"`
	Venue           string    `json:"venue"`
}

type postShowsResponse struct {
	ShowID string `json:"show_id"`
}

func (s Server) PostShows(c echo.Context) error {
	var request postShowsRequest
	err := c.Bind(&request)
	if err != nil {
		return err
	}

	showID := uuid.NewString()
	err = s.showsRepo.Store(c.Request().Context(), entity.Show{
		ShowID:          showID,
		DeadNationID:    request.DeadNationID,
		NumberOfTickets: request.NumberOfTickets,
		StartTime:       request.StartTime,
		Title:           request.Title,
		Venue:           request.Venue,
	})
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, postShowsResponse{
		ShowID: showID,
	})
}
