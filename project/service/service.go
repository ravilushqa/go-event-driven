package service

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/go-event-driven/common/log"
	"github.com/ThreeDotsLabs/watermill-redisstream/pkg/redisstream"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"

	"tickets/db"
	"tickets/db/bookings"
	"tickets/db/read_model_ops_bookings"
	"tickets/db/shows"
	"tickets/db/tickets"
	"tickets/http"
	"tickets/pubsub"
	"tickets/pubsub/bus"
	"tickets/pubsub/command"
	"tickets/pubsub/event"
	"tickets/pubsub/outbox"
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
	addr string,
	db *sqlx.DB,
	redisClient *redis.Client,
	spreadsheetsService event.SpreadsheetsAPI,
	receiptsService event.ReceiptsService,
	fileService event.FileService,
	deadNationService event.DeadNationService,
	paymentService event.PaymentService,
) Service {
	ticketsRepo := tickets.NewPostgresRepository(db)
	showsRepo := shows.NewPostgresRepository(db)
	bookingsRepo := bookings.NewPostgresRepository(db)
	opsReadModel := read_model_ops_bookings.NewOpsBookingReadModel(db)

	watermillLogger := log.NewWatermill(log.FromContext(context.Background()))

	var redisPublisher message.Publisher
	redisPublisher = pubsub.NewRedisPublisher(redisClient, watermillLogger)
	redisPublisher = log.CorrelationPublisherDecorator{Publisher: redisPublisher}

	eventBus, err := bus.NewEventBus(redisPublisher)
	if err != nil {
		panic(fmt.Errorf("failed to create event bus: %w", err))
	}

	eventsHandler := event.NewHandler(
		eventBus,
		spreadsheetsService,
		receiptsService,
		fileService,
		deadNationService,
		paymentService,
		ticketsRepo,
		showsRepo,
	)

	commandBus, err := bus.NewCommandBus(redisPublisher)
	if err != nil {
		panic(fmt.Errorf("failed to create command bus: %w", err))
	}

	commandsHandler := command.NewHandler(
		eventBus,
		receiptsService,
		paymentService,
	)

	postgresSubscriber := outbox.NewPostgresSubscriber(db.DB, watermillLogger)
	eventProcessorConfig := event.NewProcessorConfig(redisClient, watermillLogger)
	commandProcessorConfig := command.NewProcessorConfig(redisClient, watermillLogger)

	redisSubscriber, err := redisstream.NewSubscriber(redisstream.SubscriberConfig{
		Client:        redisClient,
		ConsumerGroup: "svc-tickets.events",
	}, watermillLogger)

	watermillRouter, err := pubsub.NewWatermillRouter(
		postgresSubscriber,
		redisPublisher,
		redisSubscriber,
		eventProcessorConfig,
		eventsHandler,
		commandProcessorConfig,
		commandsHandler,
		opsReadModel,
		watermillLogger,
	)
	if err != nil {
		panic(fmt.Errorf("failed to create watermill router: %w", err))
	}

	httpServer := http.NewServer(
		addr,
		eventBus,
		commandBus,
		spreadsheetsService,
		ticketsRepo,
		showsRepo,
		bookingsRepo,
		opsReadModel,
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
