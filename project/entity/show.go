package entity

import "time"

type Show struct {
	ShowID          string    `json:"show_id" db:"show_id"`
	DeadNationID    string    `json:"dead_nation_id" db:"dead_nation_id"`
	NumberOfTickets int       `json:"number_of_tickets" db:"number_of_tickets"`
	StartTime       time.Time `json:"start_time" db:"start_time"`
	Title           string    `json:"title" db:"title"`
	Venue           string    `json:"venue" db:"venue"`
}
