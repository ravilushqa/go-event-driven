package pubsub

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/entity"
)

func (h Handler) PrintTicketHandler() cqrs.EventHandler {
	return cqrs.NewEventHandler(
		"PrintTicketHandler",
		func(ctx context.Context, event *entity.TicketBookingConfirmed) error {
			log.FromContext(ctx).Info("Printing ticket")

			ticketHTML := `
			<html>
				<head>
					<title>Ticket</title>
				</head>
				<body>
					<h1>Ticket ` + event.TicketID + `</h1>
					<p>Price: ` + event.Price.Amount + ` ` + event.Price.Currency + `</p>	
				</body>
			</html>
			`

			fileID := fmt.Sprintf("%s-ticket.html", event.TicketID)
			err := h.filesService.UploadFile(ctx, fileID, ticketHTML)
			if err != nil {
				return err
			}

			ticketPrinter := entity.TicketPrinted{
				Header:   entity.NewEventHeader(),
				TicketID: event.TicketID,
				FileName: fileID,
			}

			return h.eventbus.Publish(ctx, ticketPrinter)
		},
	)
}
