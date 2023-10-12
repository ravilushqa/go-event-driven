package service

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/components/cqrs"

	"tickets/db"
	"tickets/db/bookings"
	"tickets/db/shows"
	"tickets/db/tickets"
	"tickets/handler/http"
	"tickets/handler/pubsub"
	"tickets/pkg"
	"tickets/pkg/outbox"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func init() {
	log.Init(logrus.InfoLevel)
}

type Service struct {
	db              *sqlx.DB
	watermillRouter *message.Router
	httpServer      *http.Server
}

func New(
	db *sqlx.DB,
	redisClient *redis.Client,
	spreadsheetsService pubsub.SpreadsheetsAPI,
	receiptsService pubsub.ReceiptsService,
	fileService pubsub.FileService,
	addr string,
) Service {
	ticketsRepo := tickets.NewPostgresRepository(db)
	showsRepo := shows.NewPostgresRepository(db)
	bookingsRepo := bookings.NewPostgresRepository(db)

	watermillLogger := log.NewWatermill(log.FromContext(context.Background()))

	var redisPublisher message.Publisher
	redisPublisher = pkg.NewRedisPublisher(redisClient, watermillLogger)
	redisPublisher = log.CorrelationPublisherDecorator{Publisher: redisPublisher}

	eventBus, err := pkg.NewEventBus(redisPublisher)
	if err != nil {
		panic(fmt.Errorf("failed to create event bus: %w", err))
	}

	txEventBus, err := pkg.NewEventBus(redisPublisher)

	eventsHandler := pubsub.NewHandler(
		spreadsheetsService,
		receiptsService,
		ticketsRepo,
		fileService,
		eventBus,
	)
	watermillRouter, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		panic(err)
	}
	pubsub.UseMiddlewares(watermillRouter, watermillLogger)

	err = pkg.RegisterEventHandlers(
		redisClient,
		watermillRouter,
		[]cqrs.EventHandler{
			eventsHandler.StoreTicketHandler(),
			eventsHandler.AppendToTrackerHandler(),
			eventsHandler.IssueReceiptHandler(),
			eventsHandler.CancelTicketHandler(),
			eventsHandler.DeleteTicketHandler(),
			eventsHandler.PrintTicketHandler(),
		},
		watermillLogger,
	)
	if err != nil {
		panic(err)
	}

	postgresSubscriber := outbox.NewPostgresSubscriber(db.DB, watermillLogger)
	outbox.AddForwarderHandler(postgresSubscriber, redisPublisher, watermillRouter, watermillLogger)

	httpServer := http.NewServer(
		addr,
		eventBus,
		txEventBus,
		spreadsheetsService,
		ticketsRepo,
		showsRepo,
		bookingsRepo,
	)

	return Service{
		db,
		watermillRouter,
		httpServer,
	}
}

func (s Service) Run(ctx context.Context) error {
	if err := db.InitializeDatabaseSchema(s.db); err != nil {
		return fmt.Errorf("failed to initialize database schema: %w", err)
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return s.watermillRouter.Run(ctx)
	})

	g.Go(func() error {
		// we don't want to start HTTP sferver before Watermill router (so service won't be healthy before it's ready)
		<-s.watermillRouter.Running()

		err := s.httpServer.Run(ctx)
		if err != nil {
			return err
		}

		return nil
	})

	return g.Wait()
}
