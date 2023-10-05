package main

type Task int

const (
	TaskIssueReceipt Task = iota
	TaskAppendToTracker
)

type Message struct {
	Task     Task
	TicketID string
}
