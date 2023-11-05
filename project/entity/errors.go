package entity

import "errors"

var (
	ErrNoAvailableTickets = errors.New("no available tickets")
	ErrConflict           = errors.New("conflict")
)
